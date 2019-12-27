package zng

import (
	"encoding/json"
	"errors"
	"math"
	"net"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

// A Record wraps a zeek.Record and can simultaneously represent its raw
// serialized zng form or its parsed zeek.Record form.  This duality lets us
// parse raw logs and perform fast-path operations directly on the zng data
// without having to parse the entire record.  Thus, the same code that performs
// operations on zeek data can work with either serialized data or native
// zeek.Records by accessing data via the Record methods.
type Record struct {
	Ts nano.Ts
	*Descriptor
	nonvolatile bool
	// Raw is the serialization format for zng records.  A raw value comprises a
	// sequence of zvals, one per descriptor column.  The descriptor is stored
	// outside of the raw serialization but is needed to interpret the raw values.
	Raw zval.Encoding
}

func NewRecord(d *Descriptor, ts nano.Ts, raw zval.Encoding) *Record {
	return &Record{
		Ts:          ts,
		Descriptor:  d,
		nonvolatile: true,
		Raw:         raw,
	}
}

func NewRecordNoTs(d *Descriptor, zv zval.Encoding) *Record {
	r := NewRecord(d, 0, zv)
	if d.TsCol >= 0 {
		body := r.Slice(d.TsCol)
		if body != nil {
			r.Ts, _ = zeek.DecodeTime(body)
		}
	}
	return r
}

func NewRecordCheck(d *Descriptor, ts nano.Ts, raw zval.Encoding) (*Record, error) {
	r := NewRecord(d, ts, raw)
	if err := r.TypeCheck(); err != nil {
		return nil, err
	}
	return r, nil
}

// NewControlRecord creates a control record from a byte slice.
func NewControlRecord(raw []byte) *Record {
	return &Record{
		nonvolatile: true,
		Raw:         raw,
	}
}

// NewVolatileRecord creates a record from a timestamp and a raw value
// marked volatile so that Keep() must be called to make it safe.
// This is useful for readers that allocate records whose raw body points
// into a reusable buffer allowing the scanner to filter these records
// without having their body copied to safe memory, i.e., when the scanner
// matches a record, it will call Keep() to make a safe copy.
func NewVolatileRecord(d *Descriptor, ts nano.Ts, raw zval.Encoding) *Record {
	return &Record{
		Ts:          ts,
		Descriptor:  d,
		nonvolatile: false,
		Raw:         raw,
	}
}

// NewRecordZvals creates a record from zvals.  If the descriptor has a field
// named ts, NewRecordZvals parses the corresponding zval as a time for use as
// the record's timestamp.  If the descriptor has no field named ts, the
// record's timestamp is zero.  NewRecordZvals returns an error if the number of
// descriptor columns and zvals do not agree or if parsing the ts zval fails.
func NewRecordZvals(d *Descriptor, vals ...zval.Encoding) (t *Record, err error) {
	raw, err := EncodeZvals(d, vals)
	if err != nil {
		return nil, err
	}
	var ts nano.Ts
	if col, ok := d.ColumnOfField("ts"); ok {
		var err error
		ts, err = zeek.DecodeTime(vals[col])
		if err != nil {
			return nil, err
		}
	}
	r := NewRecord(d, ts, raw)
	if err := r.TypeCheck(); err != nil {
		return nil, err
	}
	return r, nil
}

// NewRecordZeekStrings creates a record from Zeek UTF-8 strings.
func NewRecordZeekStrings(d *Descriptor, ss ...string) (t *Record, err error) {
	vals := make([][]byte, 0, 32)
	for _, s := range ss {
		vals = append(vals, []byte(s))
	}
	zv, ts, err := NewRawAndTsFromZeekValues(d, d.TsCol, vals)
	if err != nil {
		return nil, err
	}
	r := NewRecord(d, ts, zv)
	if err := r.TypeCheck(); err != nil {
		return nil, err
	}
	return r, nil
}

// ZvalIter returns an iterator over the receiver's zvals.
func (r *Record) ZvalIter() zval.Iter {
	return r.Raw.Iter()
}

// Width returns the number of columns in the record.
func (r *Record) Width() int { return len(r.Descriptor.Type.Columns) }

func (r *Record) Keep() *Record {
	if r.nonvolatile {
		return r
	}
	v := &Record{Ts: r.Ts, Descriptor: r.Descriptor, nonvolatile: true}
	v.Raw = make(zval.Encoding, len(r.Raw))
	copy(v.Raw, r.Raw)
	return v
}

func (r *Record) CopyBody() {
	if r.nonvolatile {
		return
	}
	body := make(zval.Encoding, len(r.Raw))
	copy(body, r.Raw)
	r.Raw = body
	r.nonvolatile = true
}

func (r *Record) HasField(field string) bool {
	_, ok := r.Descriptor.LUT[field]
	return ok
}

func (r *Record) Bytes() []byte {
	if r.Raw == nil {
		panic("this shouldn't happen")
	}
	return r.Raw
}

func isHighPrecision(ts nano.Ts) bool {
	_, ns := ts.Split()
	return (ns/1000)*1000 != ns
}

// This returns the zeek strings for this record.  It works only for records
// that can be represented as legacy zeek values.  XXX We need to not use this.
// XXX change to Pretty for output writers?... except zeek?
func (r *Record) ZeekStrings(precision int, utf8 bool) ([]string, bool, error) {
	var ss []string
	it := r.ZvalIter()
	var changePrecision bool
	for _, col := range r.Descriptor.Type.Columns {
		val, isContainer, err := it.Next()
		if err != nil {
			return nil, false, err
		}
		var field string
		if precision >= 0 && col.Type == zeek.TypeTime && val != nil {
			ts, err := zeek.DecodeTime(val)
			if err != nil {
				return nil, false, err
			}
			if precision == 6 && isHighPrecision(ts) {
				precision = 9
				changePrecision = true
			}
			field = string(ts.AppendFloat(nil, precision))
		} else {
			field = ZvalToZeekString(col.Type, val, isContainer, utf8)
		}
		ss = append(ss, field)
	}
	return ss, changePrecision, nil
}

// TypeCheck checks that the value coding in Raw is structurally consistent
// with this value's descriptor.  It does not check that the actual leaf
// values when parsed are type compatible with the leaf types.
func (r *Record) TypeCheck() error {
	return checkRecord(r.Descriptor.Type, r.Raw)
}

var (
	ErrMissingField = errors.New("record missing a field")
	ErrExtraField   = errors.New("record with extra field")
	ErrNotContainer = errors.New("scalar where container was expected")
	ErrNotScalar    = errors.New("container where scalar was expected")
)

func checkVector(typ *zeek.TypeVector, body zval.Encoding) error {
	if body == nil {
		return nil
	}
	inner := zeek.InnerType(typ)
	it := zval.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch v := inner.(type) {
		case *zeek.TypeRecord:
			if !container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkRecord(v, body); err != nil {
				return err
			}
		case *zeek.TypeVector:
			if !container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkVector(v, body); err != nil {
				return err
			}
		case *zeek.TypeSet:
			if !container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkSet(v, body); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotScalar}
			}
		}
	}
	return nil
}

func checkSet(typ *zeek.TypeSet, body zval.Encoding) error {
	if body == nil {
		return nil
	}
	inner := zeek.InnerType(typ)
	if zeek.IsContainerType(inner) {
		return &RecordTypeError{Name: "<set>", Type: typ.String(), Err: ErrNotScalar}
	}
	it := zval.Iter(body)
	for !it.Done() {
		_, container, err := it.Next()
		if err != nil {
			return err
		}
		if container {
			return &RecordTypeError{Name: "<set element>", Type: typ.String(), Err: ErrNotScalar}
		}
	}
	return nil
}

func checkRecord(typ *zeek.TypeRecord, body zval.Encoding) error {
	if body == nil {
		return nil
	}
	it := zval.Iter(body)
	for _, col := range typ.Columns {
		if it.Done() {
			return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrMissingField}
		}
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch v := col.Type.(type) {
		case *zeek.TypeRecord:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkRecord(v, body); err != nil {
				return err
			}
		case *zeek.TypeVector:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkVector(v, body); err != nil {
				return err
			}
		case *zeek.TypeSet:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkSet(v, body); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotScalar}
			}
		}
	}
	return nil
}

func (r *Record) ValueByColumn(col int) zeek.Value {
	//XXX shouldn't ignore error
	v, _ := r.Descriptor.Type.Columns[col].Type.New(r.Slice(col))
	return v
}

func (r *Record) ValueByField(field string) zeek.Value {
	//XXX shouldn't ignore error
	col, ok := r.ColumnOfField(field)
	if ok {
		return r.ValueByColumn(col)
	}
	return nil
}

func (r *Record) Slice(column int) zval.Encoding {
	var zv zval.Encoding
	for i, it := 0, zval.Iter(r.Raw); i <= column; i++ {
		if it.Done() {
			return nil
		}
		var err error
		zv, _, err = it.Next()
		if err != nil {
			return nil
		}
	}
	return zv
}

func (r *Record) TypedSlice(colno int) zeek.TypedEncoding {
	return zeek.TypedEncoding{
		Type: r.Descriptor.Type.Columns[colno].Type,
		Body: r.Slice(colno),
	}
}

func (r *Record) Value(colno int) zeek.Value {
	v := r.TypedSlice(colno)
	val, err := v.Type.New(v.Body)
	if err != nil {
		return nil
	}
	return val
}

func (r *Record) StringOfColumn(colno int) string {
	val := r.Value(colno)
	if val != nil {
		return ""
	}
	return val.String()
}

func (r *Record) ColumnOfField(field string) (int, bool) {
	return r.Descriptor.ColumnOfField(field)
}

func (r *Record) TypeOfColumn(col int) zeek.Type {
	return r.Descriptor.Type.Columns[col].Type
}

func (r *Record) Access(field string) (zeek.TypedEncoding, error) {
	if k, ok := r.Descriptor.LUT[field]; ok {
		typ := r.Descriptor.Type.Columns[k].Type
		v := r.Slice(k)
		return zeek.TypedEncoding{typ, v}, nil
	}
	return zeek.TypedEncoding{}, ErrNoSuchField

}

func (r *Record) AccessString(field string) (string, error) {
	e, err := r.Access(field)
	if err != nil {
		return "", err
	}
	if _, ok := e.Type.(*zeek.TypeOfString); !ok {
		return "", ErrTypeMismatch
	}
	return zeek.DecodeString(e.Body)
}

func (r *Record) AccessBool(field string) (bool, error) {
	e, err := r.Access(field)
	if err != nil {
		return false, err
	}
	if _, ok := e.Type.(*zeek.TypeOfBool); !ok {
		return false, ErrTypeMismatch
	}
	return zeek.DecodeBool(e.Body)
}

func (r *Record) AccessInt(field string) (int64, error) {
	e, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	switch e.Type.(type) {
	case *zeek.TypeOfInt:
		return zeek.DecodeInt(e.Body)
	case *zeek.TypeOfCount:
		v, err := zeek.DecodeCount(e.Body)
		if v > math.MaxInt64 {
			return 0, errors.New("conversion from type count to int results in overflow")
		}
		return int64(v), err
	case *zeek.TypeOfPort:
		v, err := zeek.DecodePort(e.Body)
		return int64(v), err
	}
	return 0, ErrTypeMismatch
}

func (r *Record) AccessDouble(field string) (float64, error) {
	e, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := e.Type.(*zeek.TypeOfDouble); !ok {
		return 0, ErrTypeMismatch
	}
	return zeek.DecodeDouble(e.Body)
}

func (r *Record) AccessIP(field string) (net.IP, error) {
	e, err := r.Access(field)
	if err != nil {
		return nil, err
	}
	if _, ok := e.Type.(*zeek.TypeOfAddr); !ok {
		return nil, ErrTypeMismatch
	}
	return zeek.DecodeAddr(e.Body)
}

func (r *Record) AccessTime(field string) (nano.Ts, error) {
	e, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := e.Type.(*zeek.TypeOfTime); !ok {
		return 0, ErrTypeMismatch
	}
	return zeek.DecodeTime(e.Body)
}

func (r *Record) AccessTimeByColumn(colno int) (nano.Ts, error) {
	zv := r.Slice(colno)
	if zv == nil {
		return 0, ErrTypeMismatch
	}
	return zeek.DecodeTime(zv)
}

func (r *Record) String() string {
	value, err := r.Descriptor.Type.New(r.Raw)
	if err != nil {
		panic(err)
	}
	return value.String()
}

// MarshalJSON implements json.Marshaler.
func (r *Record) MarshalJSON() ([]byte, error) {
	value, err := r.Descriptor.Type.New(r.Raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func Descriptors(recs []*Record) []*Descriptor {
	m := make(map[int]*Descriptor)
	for _, r := range recs {
		if r.Descriptor != nil {
			m[r.Descriptor.ID] = r.Descriptor
		}
	}
	descriptors := make([]*Descriptor, len(m))
	i := 0
	for id := range m {
		descriptors[i] = m[id]
		i++
	}
	return descriptors
}
