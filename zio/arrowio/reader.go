package arrowio

import (
	"fmt"
	"io"
	"strconv"
	"unsafe"

	"github.com/apache/arrow/go/v11/arrow"
	"github.com/apache/arrow/go/v11/arrow/array"
	"github.com/apache/arrow/go/v11/arrow/ipc"
	"github.com/apache/arrow/go/v11/parquet/pqarrow"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/exp/slices"
)

// Reader is a zio.Reader for the Arrow IPC stream format.
type Reader struct {
	zctx *zed.Context
	rr   pqarrow.RecordReader

	typ              zed.Type
	unionTagMappings map[string][]int

	rec arrow.Record
	i   int

	builder zcode.Builder
	val     zed.Value
}

func NewReader(zctx *zed.Context, r io.Reader) (*Reader, error) {
	ipcReader, err := ipc.NewReader(r)
	if err != nil {
		return nil, err
	}
	ar, err := NewReaderFromRecordReader(zctx, ipcReader)
	if err != nil {
		ipcReader.Release()
		return nil, err
	}
	return ar, nil
}

func NewReaderFromRecordReader(zctx *zed.Context, rr pqarrow.RecordReader) (*Reader, error) {
	fields := slices.Clone(rr.Schema().Fields())
	uniquifyFieldNames(fields)
	r := &Reader{
		zctx:             zctx,
		rr:               rr,
		unionTagMappings: map[string][]int{},
	}
	typ, err := r.newZedType(arrow.StructOf(fields...))
	if err != nil {
		return nil, err
	}
	r.typ = typ
	return r, nil
}

func uniquifyFieldNames(fields []arrow.Field) {
	names := map[string]int{}
	for i, f := range fields {
		if n := names[f.Name]; n > 0 {
			fields[i].Name += strconv.Itoa(n)
		}
		names[f.Name]++
	}
}

func (r *Reader) Close() error {
	if r.rr != nil {
		r.rr.Release()
		r.rr = nil
	}
	if r.rec != nil {
		r.rec.Release()
		r.rec = nil
	}
	return nil
}

func (r *Reader) Read() (*zed.Value, error) {
	for r.rec == nil {
		rec, err := r.rr.Read()
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}
		if rec.NumRows() > 0 {
			r.rec = rec
			r.i = 0
		} else {
			rec.Release()
		}
	}
	r.builder.Truncate()
	for _, array := range r.rec.Columns() {
		if err := r.buildZcode(array, r.i); err != nil {
			return nil, err
		}
	}
	r.val = *zed.NewValue(r.typ, r.builder.Bytes())
	r.i++
	if r.i >= int(r.rec.NumRows()) {
		r.rec.Release()
		r.rec = nil
	}
	return &r.val, nil
}

var dayTimeIntervalFields = []zed.Column{
	{Name: "days", Type: zed.TypeInt32},
	{Name: "milliseconds", Type: zed.TypeUint32},
}
var decimal128Fields = []zed.Column{
	{Name: "high", Type: zed.TypeInt64},
	{Name: "low", Type: zed.TypeUint64},
}
var monthDayNanoIntervalFields = []zed.Column{
	{Name: "month", Type: zed.TypeInt32},
	{Name: "day", Type: zed.TypeInt32},
	{Name: "nanoseconds", Type: zed.TypeInt64},
}

func (r *Reader) newZedType(dt arrow.DataType) (zed.Type, error) {
	// Order here follows that of the arrow.Time constants.
	switch dt.ID() {
	case arrow.NULL:
		return zed.TypeNull, nil
	case arrow.BOOL:
		return zed.TypeBool, nil
	case arrow.UINT8:
		return zed.TypeUint8, nil
	case arrow.INT8:
		return zed.TypeInt8, nil
	case arrow.UINT16:
		return zed.TypeUint16, nil
	case arrow.INT16:
		return zed.TypeInt16, nil
	case arrow.UINT32:
		return zed.TypeUint32, nil
	case arrow.INT32:
		return zed.TypeInt32, nil
	case arrow.UINT64:
		return zed.TypeUint64, nil
	case arrow.INT64:
		return zed.TypeInt64, nil
	case arrow.FLOAT16:
		return zed.TypeFloat16, nil
	case arrow.FLOAT32:
		return zed.TypeFloat32, nil
	case arrow.FLOAT64:
		return zed.TypeFloat64, nil
	case arrow.STRING:
		return zed.TypeString, nil
	case arrow.BINARY:
		return zed.TypeBytes, nil
	case arrow.FIXED_SIZE_BINARY:
		width := strconv.Itoa(dt.(*arrow.FixedSizeBinaryType).ByteWidth)
		return r.zctx.LookupTypeNamed("arrow_fixed_size_binary_"+width, zed.TypeBytes)
	case arrow.DATE32:
		return r.zctx.LookupTypeNamed("arrow_date32", zed.TypeTime)
	case arrow.DATE64:
		return r.zctx.LookupTypeNamed("arrow_date64", zed.TypeTime)
	case arrow.TIMESTAMP:
		if unit := dt.(*arrow.TimestampType).Unit; unit != arrow.Nanosecond {
			return r.zctx.LookupTypeNamed("arrow_timestamp_"+unit.String(), zed.TypeTime)
		}
		return zed.TypeTime, nil
	case arrow.TIME32:
		unit := dt.(*arrow.Time32Type).Unit.String()
		return r.zctx.LookupTypeNamed("arrow_time32_"+unit, zed.TypeTime)
	case arrow.TIME64:
		unit := dt.(*arrow.Time64Type).Unit.String()
		return r.zctx.LookupTypeNamed("arrow_time64_"+unit, zed.TypeTime)
	case arrow.INTERVAL_MONTHS:
		return r.zctx.LookupTypeNamed("arrow_month_interval", zed.TypeInt32)
	case arrow.INTERVAL_DAY_TIME:
		typ, err := r.zctx.LookupTypeRecord(dayTimeIntervalFields)
		if err != nil {
			return nil, err
		}
		return r.zctx.LookupTypeNamed("arrow_day_time_interval", typ)
	case arrow.DECIMAL128:
		typ, err := r.zctx.LookupTypeRecord(decimal128Fields)
		if err != nil {
			return nil, err
		}
		return r.zctx.LookupTypeNamed("arrow_decimal128", typ)
	case arrow.DECIMAL256:
		return r.zctx.LookupTypeNamed("arrow_decimal256", r.zctx.LookupTypeArray(zed.TypeUint64))
	case arrow.LIST:
		typ, err := r.newZedType(dt.(*arrow.ListType).Elem())
		if err != nil {
			return nil, err
		}
		return r.zctx.LookupTypeArray(typ), nil
	case arrow.STRUCT:
		var fields []zed.Column
		for _, f := range dt.(*arrow.StructType).Fields() {
			typ, err := r.newZedType(f.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, zed.NewColumn(f.Name, typ))
		}
		return r.zctx.LookupTypeRecord(fields)
	case arrow.SPARSE_UNION, arrow.DENSE_UNION:
		return r.newZedUnionType(dt.(arrow.UnionType), dt.Fingerprint())
	case arrow.DICTIONARY:
		return r.newZedType(dt.(*arrow.DictionaryType).ValueType)
	case arrow.MAP:
		keyType, err := r.newZedType(dt.(*arrow.MapType).KeyType())
		if err != nil {
			return nil, err
		}
		itemType, err := r.newZedType(dt.(*arrow.MapType).ItemType())
		if err != nil {
			return nil, err
		}
		return r.zctx.LookupTypeMap(keyType, itemType), nil
	case arrow.FIXED_SIZE_LIST:
		typ, err := r.newZedType(dt.(*arrow.FixedSizeListType).Elem())
		if err != nil {
			return nil, err
		}
		size := strconv.Itoa(int(dt.(*arrow.FixedSizeListType).Len()))
		return r.zctx.LookupTypeNamed("arrow_fixed_size_list_"+size, r.zctx.LookupTypeArray(typ))
	case arrow.DURATION:
		if unit := dt.(*arrow.DurationType).Unit; unit != arrow.Nanosecond {
			return r.zctx.LookupTypeNamed("arrow_duration_"+unit.String(), zed.TypeDuration)
		}
		return zed.TypeDuration, nil
	case arrow.LARGE_STRING:
		return r.zctx.LookupTypeNamed("arrow_large_string", zed.TypeString)
	case arrow.LARGE_BINARY:
		return r.zctx.LookupTypeNamed("arrow_large_binary", zed.TypeBytes)
	case arrow.LARGE_LIST:
		typ, err := r.newZedType(dt.(*arrow.LargeListType).Elem())
		if err != nil {
			return nil, err
		}
		return r.zctx.LookupTypeNamed("arrow_large_list", r.zctx.LookupTypeArray(typ))
	case arrow.INTERVAL_MONTH_DAY_NANO:
		typ, err := r.zctx.LookupTypeRecord(monthDayNanoIntervalFields)
		if err != nil {
			return nil, err
		}
		return r.zctx.LookupTypeNamed("arrow_month_day_nano_interval", typ)
	default:
		return nil, fmt.Errorf("unimplemented Arrow type: %s", dt.Name())
	}
}

func (r *Reader) newZedUnionType(union arrow.UnionType, fingerprint string) (zed.Type, error) {
	var types []zed.Type
	for _, f := range union.Fields() {
		typ, err := r.newZedType(f.Type)
		if err != nil {
			return nil, err
		}
		types = append(types, typ)
	}
	uniqueTypes := zed.UniqueTypes(slices.Clone(types))
	var x []int
Loop:
	for _, typ2 := range types {
		for i, typ := range uniqueTypes {
			if typ == typ2 {
				x = append(x, i)
				continue Loop
			}
		}
	}
	r.unionTagMappings[fingerprint] = x
	return r.zctx.LookupTypeUnion(uniqueTypes), nil
}

func (r *Reader) buildZcode(a arrow.Array, i int) error {
	b := &r.builder
	if a.IsNull(i) {
		b.Append(nil)
		return nil
	}
	data := a.Data()
	// XXX Calling array.New*Data once per value (rather than once
	// per arrow.Array) is slow.
	//
	// Order here follows that of the arrow.Time constants.
	switch a.DataType().ID() {
	case arrow.NULL:
		b.Append(nil)
	case arrow.BOOL:
		b.Append(zed.EncodeBool(array.NewBooleanData(data).Value(i)))
	case arrow.UINT8:
		b.Append(zed.EncodeUint(uint64(array.NewUint8Data(data).Value(i))))
	case arrow.INT8:
		b.Append(zed.EncodeInt(int64(array.NewInt8Data(data).Value(i))))
	case arrow.UINT16:
		b.Append(zed.EncodeUint(uint64(array.NewUint16Data(data).Value(i))))
	case arrow.INT16:
		b.Append(zed.EncodeInt(int64(array.NewInt16Data(data).Value(i))))
	case arrow.UINT32:
		b.Append(zed.EncodeUint(uint64(array.NewUint32Data(data).Value(i))))
	case arrow.INT32:
		b.Append(zed.EncodeInt(int64(array.NewInt32Data(data).Value(i))))
	case arrow.UINT64:
		b.Append(zed.EncodeUint(array.NewUint64Data(data).Value(i)))
	case arrow.INT64:
		b.Append(zed.EncodeInt(array.NewInt64Data(data).Value(i)))
	case arrow.FLOAT16:
		b.Append(zed.EncodeFloat16(array.NewFloat16Data(data).Value(i).Float32()))
	case arrow.FLOAT32:
		b.Append(zed.EncodeFloat32(array.NewFloat32Data(data).Value(i)))
	case arrow.FLOAT64:
		b.Append(zed.EncodeFloat64(array.NewFloat64Data(data).Value(i)))
	case arrow.STRING:
		appendString(b, array.NewStringData(data).Value(i))
	case arrow.BINARY:
		b.Append(zed.EncodeBytes(array.NewBinaryData(data).Value(i)))
	case arrow.FIXED_SIZE_BINARY:
		b.Append(zed.EncodeBytes(array.NewFixedSizeBinaryData(data).Value(i)))
	case arrow.DATE32:
		b.Append(zed.EncodeTime(nano.TimeToTs(array.NewDate32Data(data).Value(i).ToTime())))
	case arrow.DATE64:
		b.Append(zed.EncodeTime(nano.TimeToTs(array.NewDate64Data(data).Value(i).ToTime())))
	case arrow.TIMESTAMP:
		unit := a.DataType().(*arrow.TimestampType).Unit
		b.Append(zed.EncodeTime(nano.TimeToTs(array.NewTimestampData(data).Value(i).ToTime(unit))))
	case arrow.TIME32:
		unit := a.DataType().(*arrow.Time32Type).Unit
		b.Append(zed.EncodeTime(nano.TimeToTs(array.NewTime32Data(data).Value(i).ToTime(unit))))
	case arrow.TIME64:
		unit := a.DataType().(*arrow.Time64Type).Unit
		b.Append(zed.EncodeTime(nano.TimeToTs(array.NewTime64Data(data).Value(i).ToTime(unit))))
	case arrow.INTERVAL_MONTHS:
		b.Append(zed.EncodeInt(int64(array.NewMonthIntervalData(data).Value(i))))
	case arrow.INTERVAL_DAY_TIME:
		v := array.NewDayTimeIntervalData(data).Value(i)
		b.BeginContainer()
		b.Append(zed.EncodeInt(int64(v.Days)))
		b.Append(zed.EncodeInt(int64(v.Milliseconds)))
		b.EndContainer()
	case arrow.DECIMAL128:
		v := array.NewDecimal128Data(data).Value(i)
		b.BeginContainer()
		b.Append(zed.EncodeInt(v.HighBits()))
		b.Append(zed.EncodeUint(v.LowBits()))
		b.EndContainer()
	case arrow.DECIMAL256:
		b.BeginContainer()
		for _, u := range array.NewDecimal256Data(data).Value(i).Array() {
			b.Append(zed.EncodeUint(u))
		}
		b.EndContainer()
	case arrow.LIST:
		v := array.NewListData(data)
		start, end := v.ValueOffsets(i)
		return r.buildZcodeList(v.ListValues(), int(start), int(end))
	case arrow.STRUCT:
		v := array.NewStructData(data)
		b.BeginContainer()
		for j := 0; j < v.NumField(); j++ {
			if err := r.buildZcode(v.Field(j), i); err != nil {
				return err
			}
		}
		b.EndContainer()
	case arrow.SPARSE_UNION:
		return r.buildZcodeUnion(array.NewSparseUnionData(data), data.DataType(), i)
	case arrow.DENSE_UNION:
		return r.buildZcodeUnion(array.NewDenseUnionData(data), data.DataType(), i)
	case arrow.DICTIONARY:
		v := array.NewDictionaryData(data)
		return r.buildZcode(v.Dictionary(), v.GetValueIndex(i))
	case arrow.MAP:
		v := array.NewMapData(data)
		keys, items := v.Keys(), v.Items()
		b.BeginContainer()
		for j, end := v.ValueOffsets(i); j < end; j++ {
			if err := r.buildZcode(keys, int(j)); err != nil {
				return err
			}
			if err := r.buildZcode(items, int(j)); err != nil {
				return err
			}
		}
		b.TransformContainer(zed.NormalizeMap)
		b.EndContainer()
	case arrow.FIXED_SIZE_LIST:
		v := array.NewFixedSizeListData(data)
		return r.buildZcodeList(v.ListValues(), 0, v.Len())
	case arrow.DURATION:
		d := nano.Duration(array.NewDurationData(data).Value(i))
		switch a.DataType().(*arrow.DurationType).Unit {
		case arrow.Second:
			d *= nano.Second
		case arrow.Millisecond:
			d *= nano.Millisecond
		case arrow.Microsecond:
			d *= nano.Microsecond
		}
		b.Append(zed.EncodeDuration(d))
	case arrow.LARGE_STRING:
		appendString(b, array.NewLargeStringData(data).Value(i))
	case arrow.LARGE_BINARY:
		b.Append(zed.EncodeBytes(array.NewLargeBinaryData(data).Value(i)))
	case arrow.LARGE_LIST:
		v := array.NewLargeListData(data)
		start, end := v.ValueOffsets(i)
		return r.buildZcodeList(v.ListValues(), int(start), int(end))
	case arrow.INTERVAL_MONTH_DAY_NANO:
		v := array.NewMonthDayNanoIntervalData(data).Value(i)
		b.BeginContainer()
		b.Append(zed.EncodeInt(int64(v.Months)))
		b.Append(zed.EncodeInt(int64(v.Days)))
		b.Append(zed.EncodeInt(int64(v.Nanoseconds)))
		b.EndContainer()
	default:
		return fmt.Errorf("unimplemented Arrow type: %s", a.DataType().Name())
	}
	return nil
}

func (r *Reader) buildZcodeList(a arrow.Array, start, end int) error {
	r.builder.BeginContainer()
	for i := start; i < end; i++ {
		if err := r.buildZcode(a, i); err != nil {
			return err
		}
	}
	r.builder.EndContainer()
	return nil
}

func (r *Reader) buildZcodeUnion(u array.Union, dt arrow.DataType, i int) error {
	childID := u.ChildID(i)
	if u, ok := u.(*array.DenseUnion); ok {
		i = int(u.ValueOffset(i))
	}
	b := &r.builder
	if field := u.Field(childID); field.IsNull(i) {
		b.Append(nil)
	} else {
		b.BeginContainer()
		b.Append(zed.EncodeInt(int64(r.unionTagMappings[dt.Fingerprint()][childID])))
		if err := r.buildZcode(field, i); err != nil {
			return err
		}
		b.EndContainer()
	}
	return nil
}

func appendString(b *zcode.Builder, s string) {
	if s == "" {
		b.Append(zed.EncodeString(s))
	} else {
		// Avoid a call to runtime.stringtoslicebyte.
		b.Append(*(*[]byte)(unsafe.Pointer(&s)))
	}
}
