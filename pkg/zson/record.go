package zson

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

// Errors...
var (
	ErrNoSuchField = errors.New("no such field")

	ErrCorruptTd = errors.New("corrupt type descriptor")

	ErrCorruptColumns = errors.New("wrong number of columns in record value")

	ErrTypeMismatch = errors.New("type retrieved does not match type requested")
)

// A Record wraps a zeek.Record and can simultaneously represent its raw
// serialized zson form or its parsed zeek.Record form.  This duality lets us
// parse raw logs and perform fast-path operations directly on the zson data
// without having to parse the entire record.  Thus, the same code that performs
// operations on zeek data can work with either serialized data or native
// zeek.Records by accessing data via the Record methods.
type Record struct {
	Ts nano.Ts
	*Descriptor
	nonvolatile bool
	ctrl        bool
	// Raw is the serialization format for zson records.  A raw value comprises a
	// sequence of zvals, one per descriptor column.  The descriptor is stored
	// outside of the raw serialization but is needed to interpret the raw values.
	Raw zval.Encoding
}

// NewRecord creates a record from a timestamp and a raw value.
func NewRecord(d *Descriptor, ts nano.Ts, raw zval.Encoding) *Record {
	return &Record{
		Ts:          ts,
		Descriptor:  d,
		nonvolatile: true,
		Raw:         raw,
	}
}

// NewControlRecord creates a control record from a byte slice.
func NewControlRecord(raw []byte) *Record {
	return &Record{
		nonvolatile: true,
		ctrl:        true,
		Raw:         raw,
	}
}

func (r *Record) IsControl() bool {
	return r.ctrl
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
func NewRecordZvals(d *Descriptor, vals ...[]byte) (t *Record, err error) {
	raw, err := NewRawFromZvals(d, vals)
	if err != nil {
		return nil, err
	}
	var ts nano.Ts
	if col, ok := d.ColumnOfField("ts"); ok {
		var err error
		ts, err = nano.Parse(vals[col])
		if err != nil {
			return nil, err
		}
	}
	return NewRecord(d, ts, raw), nil
}

// NewRecordZeekStrings creates a record from Zeek UTF-8 strings.
func NewRecordZeekStrings(d *Descriptor, ss ...string) (t *Record, err error) {
	tsCol, ok := d.ColumnOfField("ts")
	if !ok {
		tsCol = -1
	}
	vals := make([][]byte, 0, 32)
	for _, s := range ss {
		vals = append(vals, []byte(s))
	}
	raw, ts, err := NewRawAndTsFromZeekValues(d, tsCol, vals)
	if err != nil {
		return nil, err
	}
	return NewRecord(d, ts, raw), nil
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

func (r *Record) Strings() ([]string, error) {
	var ss []string
	it := r.ZvalIter()
	for _, col := range r.Descriptor.Type.Columns {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		ss = append(ss, ZvalToZeekString(col.Type, val))
	}
	return ss, nil
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

func (r *Record) Slice(column int) []byte {
	var val []byte
	for i, it := 0, r.ZvalIter(); i <= column; i++ {
		if it.Done() {
			return nil
		}
		var err error
		val, _, err = it.Next()
		if err != nil {
			return nil
		}
	}
	return val
}

func (r *Record) String(column int) string {
	return string(r.Slice(column))
}

func (r *Record) ColumnOfField(field string) (int, bool) {
	return r.Descriptor.ColumnOfField(field)
}

func (r *Record) TypeOfColumn(col int) zeek.Type {
	return r.Descriptor.Type.Columns[col].Type
}

func (r *Record) Access(field string) ([]byte, zeek.Type, error) {
	if k, ok := r.Descriptor.LUT[field]; ok {
		return r.Slice(k), r.Descriptor.Type.Columns[k].Type, nil
	}
	return nil, nil, ErrNoSuchField

}

func (r *Record) AccessString(field string) (string, error) {
	b, typ, err := r.Access(field)
	if err != nil {
		return "", err
	}
	typeString, ok := typ.(*zeek.TypeOfString)
	if !ok {
		return "", ErrTypeMismatch
	}
	return typeString.Parse(b)
}

func (r *Record) AccessBool(field string) (bool, error) {
	b, typ, err := r.Access(field)
	if err != nil {
		return false, err
	}
	typeBool, ok := typ.(*zeek.TypeOfBool)
	if !ok {
		return false, ErrTypeMismatch
	}
	return typeBool.Parse(b)
}

func (r *Record) AccessInt(field string) (int64, error) {
	b, typ, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	switch typ := typ.(type) {
	case *zeek.TypeOfInt:
		return typ.Parse(b)
	case *zeek.TypeOfCount:
		v, err := typ.Parse(b)
		return int64(v), err
	case *zeek.TypeOfPort:
		v, err := typ.Parse(b)
		return int64(v), err
	}
	return 0, ErrTypeMismatch
}

func (r *Record) AccessDouble(field string) (float64, error) {
	b, typ, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	typeDouble, ok := typ.(*zeek.TypeOfDouble)
	if !ok {
		return 0, ErrTypeMismatch
	}
	return typeDouble.Parse(b)
}

func (r *Record) AccessIP(field string) (net.IP, error) {
	b, typ, err := r.Access(field)
	if err != nil {
		return nil, err
	}
	typeAddr, ok := typ.(*zeek.TypeOfAddr)
	if !ok {
		return nil, ErrTypeMismatch
	}
	return typeAddr.Parse(b)
}

func (r *Record) AccessTime(field string) (nano.Ts, error) {
	b, typ, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	typeTime, ok := typ.(*zeek.TypeOfTime)
	if !ok {
		return 0, ErrTypeMismatch
	}
	return typeTime.Parse(b)
}

// Cut returns a slice of the receiver's raw values for the requested fields.
// Note that the raw values must be copied if they will be used after the
// receiver's Buffer is reclaimed.  If dest's underlying array is large enough,
// Cut uses it for the returned slice.  Otherwise, a new array is allocated.
// Cut returns nil if any field is missing from the receiver.
func (r *Record) Cut(fields []string, dest [][]byte) [][]byte {
	if n := len(fields); cap(dest) < n {
		dest = make([][]byte, n)
	} else {
		dest = dest[:n]
	}
	for k, field := range fields {
		if col, ok := r.Descriptor.LUT[field]; ok {
			dest[k] = r.Slice(col)
		} else {
			return nil
		}
	}
	return dest
}

// second return value is a bitmap of which fields were found
// XXX will not work properly if cutting >64 columns
func (r *Record) CutTypes(fields []string) ([]zeek.Type, uint64) {
	var found uint64
	valid := true
	n := len(fields)
	cut := make([]zeek.Type, n)
	for k, field := range fields {
		if col, ok := r.Descriptor.LUT[field]; ok {
			cut[k] = r.Descriptor.Type.Columns[col].Type
			found |= 1 << k
		} else {
			valid = false
		}
	}
	if valid {
		return cut, found
	}
	return nil, found
}

// MarshalJSON implements json.Marshaler.
func (r *Record) MarshalJSON() ([]byte, error) {
	value, err := r.Descriptor.Type.New(r.ZvalIter())
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
