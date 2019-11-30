package zson

import (
	"encoding/json"
	"net"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
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
	Channel     uint16
	// Raw is the serialization format for zson records.  A raw value comprises a
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
			r.Ts, _ = zeek.TypeTime.Parse(body)
		}
	}
	return r
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
func NewRecordZvals(d *Descriptor, vals ...zval.Encoding) (t *Record, err error) {
	raw, err := EncodeZvals(d, vals)
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
	vals := make([][]byte, 0, 32)
	for _, s := range ss {
		vals = append(vals, []byte(s))
	}
	zv, ts, err := NewRawAndTsFromZeekValues(d, d.TsCol, vals)
	if err != nil {
		return nil, err
	}
	return NewRecord(d, ts, zv), nil
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
	v := &Record{Ts: r.Ts, Descriptor: r.Descriptor, nonvolatile: true, Channel: r.Channel}
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

// This returns the zeek strings for this record.  It works only for records
// that can be represented as legacy zeek values.  XXX We need to not use this.
// XXX change to Pretty for output writers?... except zeek?
func (r *Record) ZeekStrings() ([]string, error) {
	var ss []string
	it := r.ZvalIter()
	for _, col := range r.Descriptor.Type.Columns {
		val, isContainer, err := it.Next()
		if err != nil {
			return nil, err
		}
		ss = append(ss, ZvalToZeekString(col.Type, val, isContainer))
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

func (r *Record) String(column int) string {
	return string(r.Slice(column))
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
	typeString, ok := e.Type.(*zeek.TypeOfString)
	if !ok {
		return "", ErrTypeMismatch
	}
	return typeString.Parse(e.Body)
}

func (r *Record) AccessBool(field string) (bool, error) {
	e, err := r.Access(field)
	if err != nil {
		return false, err
	}
	typeBool, ok := e.Type.(*zeek.TypeOfBool)
	if !ok {
		return false, ErrTypeMismatch
	}
	return typeBool.Parse(e.Body)
}

func (r *Record) AccessInt(field string) (int64, error) {
	e, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	switch typ := e.Type.(type) {
	case *zeek.TypeOfInt:
		return typ.Parse(e.Body)
	case *zeek.TypeOfCount:
		v, err := typ.Parse(e.Body)
		return int64(v), err
	case *zeek.TypeOfPort:
		v, err := typ.Parse(e.Body)
		return int64(v), err
	}
	return 0, ErrTypeMismatch
}

func (r *Record) AccessDouble(field string) (float64, error) {
	e, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	typeDouble, ok := e.Type.(*zeek.TypeOfDouble)
	if !ok {
		return 0, ErrTypeMismatch
	}
	return typeDouble.Parse(e.Body)
}

func (r *Record) AccessIP(field string) (net.IP, error) {
	e, err := r.Access(field)
	if err != nil {
		return nil, err
	}
	typeAddr, ok := e.Type.(*zeek.TypeOfAddr)
	if !ok {
		return nil, ErrTypeMismatch
	}
	return typeAddr.Parse(e.Body)
}

func (r *Record) AccessTime(field string) (nano.Ts, error) {
	e, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	typeTime, ok := e.Type.(*zeek.TypeOfTime)
	if !ok {
		return 0, ErrTypeMismatch
	}
	return typeTime.Parse(e.Body)
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
