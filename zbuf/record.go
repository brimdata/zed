package zbuf

import (
	"errors"
	"math"
	"net"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

// A Record wraps a zng.Record and can simultaneously represent its raw
// serialized zng form or its parsed zng.Record form.  This duality lets us
// parse raw logs and perform fast-path operations directly on the zng data
// without having to parse the entire record.  Thus, the same code that performs
// operations on zeek data can work with either serialized data or native
// zng.Records by accessing data via the Record methods.
type Record struct {
	Ts nano.Ts
	*Descriptor
	nonvolatile bool
	// Raw is the serialization format for zng records.  A raw value comprises a
	// sequence of zvals, one per descriptor column.  The descriptor is stored
	// outside of the raw serialization but is needed to interpret the raw values.
	Raw zcode.Bytes
}

func NewRecord(d *Descriptor, ts nano.Ts, raw zcode.Bytes) *Record {
	return &Record{
		Ts:          ts,
		Descriptor:  d,
		nonvolatile: true,
		Raw:         raw,
	}
}

func NewRecordNoTs(d *Descriptor, zv zcode.Bytes) *Record {
	r := NewRecord(d, 0, zv)
	if d.TsCol >= 0 {
		body, err := r.Slice(d.TsCol)
		if err == nil {
			r.Ts, _ = zng.DecodeTime(body)
		}
	}
	return r
}

func NewRecordCheck(d *Descriptor, ts nano.Ts, raw zcode.Bytes) (*Record, error) {
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
func NewVolatileRecord(d *Descriptor, ts nano.Ts, raw zcode.Bytes) *Record {
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
func NewRecordZvals(d *Descriptor, vals ...zcode.Bytes) (t *Record, err error) {
	raw, err := EncodeZvals(d, vals)
	if err != nil {
		return nil, err
	}
	var ts nano.Ts
	if col, ok := d.ColumnOfField("ts"); ok {
		var err error
		ts, err = zng.DecodeTime(vals[col])
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
func (r *Record) ZvalIter() zcode.Iter {
	return r.Raw.Iter()
}

// Width returns the number of columns in the record.
func (r *Record) Width() int { return len(r.Descriptor.Type.Columns) }

func (r *Record) Keep() *Record {
	if r.nonvolatile {
		return r
	}
	v := &Record{Ts: r.Ts, Descriptor: r.Descriptor, nonvolatile: true}
	v.Raw = make(zcode.Bytes, len(r.Raw))
	copy(v.Raw, r.Raw)
	return v
}

func (r *Record) CopyBody() {
	if r.nonvolatile {
		return
	}
	body := make(zcode.Bytes, len(r.Raw))
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
		val, _, err := it.Next()
		if err != nil {
			return nil, false, err
		}
		var field string
		if precision >= 0 && col.Type == zng.TypeTime && val != nil {
			ts, err := zng.DecodeTime(val)
			if err != nil {
				return nil, false, err
			}
			if precision == 6 && isHighPrecision(ts) {
				precision = 9
				changePrecision = true
			}
			field = string(ts.AppendFloat(nil, precision))
		} else {
			field = ZvalToZeekString(col.Type, val, utf8)
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
	ErrNotContainer = errors.New("expected container type, got primitive")
	ErrNotPrimitive = errors.New("expected primitive type, got container")
)

func checkVector(typ *zng.TypeVector, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	inner := zng.InnerType(typ)
	it := zcode.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch v := inner.(type) {
		case *zng.TypeRecord:
			if !container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkRecord(v, body); err != nil {
				return err
			}
		case *zng.TypeVector:
			if !container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkVector(v, body); err != nil {
				return err
			}
		case *zng.TypeSet:
			if !container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkSet(v, body); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: "<vector element>", Type: v.String(), Err: ErrNotPrimitive}
			}
		}
	}
	return nil
}

func checkSet(typ *zng.TypeSet, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	inner := zng.InnerType(typ)
	if zng.IsContainerType(inner) {
		return &RecordTypeError{Name: "<set>", Type: typ.String(), Err: ErrNotPrimitive}
	}
	it := zcode.Iter(body)
	for !it.Done() {
		_, container, err := it.Next()
		if err != nil {
			return err
		}
		if container {
			return &RecordTypeError{Name: "<set element>", Type: typ.String(), Err: ErrNotPrimitive}
		}
	}
	return nil
}

func checkRecord(typ *zng.TypeRecord, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	it := zcode.Iter(body)
	for _, col := range typ.Columns {
		if it.Done() {
			return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrMissingField}
		}
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch v := col.Type.(type) {
		case *zng.TypeRecord:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkRecord(v, body); err != nil {
				return err
			}
		case *zng.TypeVector:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkVector(v, body); err != nil {
				return err
			}
		case *zng.TypeSet:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkSet(v, body); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotPrimitive}
			}
		}
	}
	return nil
}

// Slice returns the encoded zcode.Bytes corresponding to the indicated
// column or an error if a problem was encountered.  If the encoded bytes
// result is nil without error, then that columnn is unset in this record value.
func (r *Record) Slice(column int) (zcode.Bytes, error) {
	var zv zcode.Bytes
	for i, it := 0, zcode.Iter(r.Raw); i <= column; i++ {
		if it.Done() {
			return nil, ErrNoSuchColumn
		}
		var err error
		zv, _, err = it.Next()
		if err != nil {
			return nil, err
		}
	}
	return zv, nil
}

// Value returns the indicated column as a zng.Value.  If the column doesn't
// exist or another error occurs, the nil Value is returned.
func (r *Record) Value(col int) zng.Value {
	zv, err := r.Slice(col)
	if err != nil {
		return zng.Value{}
	}
	return zng.Value{r.Descriptor.Type.Columns[col].Type, zv}
}

func (r *Record) ValueByField(field string) (zng.Value, error) {
	col, ok := r.ColumnOfField(field)
	if !ok {
		return zng.Value{}, ErrNoSuchField
	}
	return r.Value(col), nil
}

func (r *Record) ColumnOfField(field string) (int, bool) {
	return r.Descriptor.ColumnOfField(field)
}

func (r *Record) TypeOfColumn(col int) zng.Type {
	return r.Descriptor.Type.Columns[col].Type
}

func (r *Record) Access(field string) (zng.Value, error) {
	col, ok := r.ColumnOfField(field)
	if !ok {
		return zng.Value{}, ErrNoSuchField
	}
	return r.Value(col), nil
}

func (r *Record) AccessString(field string) (string, error) {
	v, err := r.Access(field)
	if err != nil {
		return "", err
	}
	if _, ok := v.Type.(*zng.TypeOfString); !ok {
		return "", ErrTypeMismatch
	}
	return zng.DecodeString(v.Bytes)
}

func (r *Record) AccessBool(field string) (bool, error) {
	v, err := r.Access(field)
	if err != nil {
		return false, err
	}
	if _, ok := v.Type.(*zng.TypeOfBool); !ok {
		return false, ErrTypeMismatch
	}
	return zng.DecodeBool(v.Bytes)
}

func (r *Record) AccessInt(field string) (int64, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	switch v.Type.(type) {
	case *zng.TypeOfInt:
		return zng.DecodeInt(v.Bytes)
	case *zng.TypeOfCount:
		v, err := zng.DecodeCount(v.Bytes)
		if v > math.MaxInt64 {
			return 0, errors.New("conversion from type count to int results in overflow")
		}
		return int64(v), err
	case *zng.TypeOfPort:
		v, err := zng.DecodePort(v.Bytes)
		return int64(v), err
	}
	return 0, ErrTypeMismatch
}

func (r *Record) AccessDouble(field string) (float64, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := v.Type.(*zng.TypeOfDouble); !ok {
		return 0, ErrTypeMismatch
	}
	return zng.DecodeDouble(v.Bytes)
}

func (r *Record) AccessIP(field string) (net.IP, error) {
	v, err := r.Access(field)
	if err != nil {
		return nil, err
	}
	if _, ok := v.Type.(*zng.TypeOfAddr); !ok {
		return nil, ErrTypeMismatch
	}
	return zng.DecodeAddr(v.Bytes)
}

func (r *Record) AccessTime(field string) (nano.Ts, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := v.Type.(*zng.TypeOfTime); !ok {
		return 0, ErrTypeMismatch
	}
	return zng.DecodeTime(v.Bytes)
}

func (r *Record) AccessTimeByColumn(colno int) (nano.Ts, error) {
	zv, err := r.Slice(colno)
	if err != nil {
		return 0, err
	}
	return zng.DecodeTime(zv)
}

func (r *Record) String() string {
	return zng.Value{r.Descriptor.Type, r.Raw}.String()
}

// MarshalJSON implements json.Marshaler.
func (r *Record) MarshalJSON() ([]byte, error) {
	// XXX zbuf.Record will get merged in with zng.Record
	return zng.Value{r.Descriptor.Type, r.Raw}.MarshalJSON()
}

//XXX
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
