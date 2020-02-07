package zng

import (
	"errors"
	"math"
	"net"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
)

var (
	ErrMissingField      = errors.New("record missing a field")
	ErrExtraField        = errors.New("record with extra field")
	ErrNotContainer      = errors.New("expected container type, got primitive")
	ErrNotPrimitive      = errors.New("expected primitive type, got container")
	ErrDescriptorExists  = errors.New("zng descriptor exists")
	ErrDescriptorInvalid = errors.New("zng descriptor out of range")
	ErrBadValue          = errors.New("malformed zng value")
	ErrBadFormat         = errors.New("malformed zng record")
	ErrTypeMismatch      = errors.New("type/value mismatch")
	ErrNoSuchField       = errors.New("no such field in zng record")
	ErrNoSuchColumn      = errors.New("no such column in zng record")
	ErrColumnMismatch    = errors.New("zng record mismatch between columns in type and columns in value")
	ErrCorruptTd         = errors.New("corrupt type descriptor")
	ErrCorruptColumns    = errors.New("wrong number of columns in zng record value")
)

type RecordTypeError struct {
	Name string
	Type string
	Err  error
}

func (r *RecordTypeError) Error() string { return r.Name + " (" + r.Type + "): " + r.Err.Error() }
func (r *RecordTypeError) Unwrap() error { return r.Err }

// XXX A Record wraps a zng.Record and can simultaneously represent its raw
// serialized zng form or its parsed zng.Record form.  This duality lets us
// parse raw logs and perform fast-path operations directly on the zng data
// without having to parse the entire record.  Thus, the same code that performs
// operations on zeek data can work with either serialized data or native
// zng.Records by accessing data via the Record methods.
type Record struct {
	Ts          nano.Ts
	Type        *TypeRecord
	nonvolatile bool
	// Raw is the serialization format for records.  A raw value comprises a
	// sequence of zvals, one per descriptor column.  The descriptor is stored
	// outside of the raw serialization but is needed to interpret the raw values.
	Raw zcode.Bytes
}

func NewRecordTs(typ *TypeRecord, ts nano.Ts, raw zcode.Bytes) *Record {
	return &Record{
		Ts:          ts,
		Type:        typ,
		nonvolatile: true,
		Raw:         raw,
	}
}

func NewRecord(typ *TypeRecord, zv zcode.Bytes) (*Record, error) {
	r := NewRecordTs(typ, 0, zv)
	if typ.TsCol < 0 {
		return r, nil
	}
	body, err := r.Slice(typ.TsCol)
	if err != nil {
		return nil, err
	}
	if body != nil {
		r.Ts, err = DecodeTime(body)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func NewRecordCheck(typ *TypeRecord, ts nano.Ts, raw zcode.Bytes) (*Record, error) {
	r := NewRecordTs(typ, ts, raw)
	if err := r.TypeCheck(); err != nil {
		return nil, err
	}
	return r, nil
}

// NewVolatileRecord creates a record from a timestamp and a raw value
// marked volatile so that Keep() must be called to make it safe.
// This is useful for readers that allocate records whose raw body points
// into a reusable buffer allowing the scanner to filter these records
// without having their body copied to safe memory, i.e., when the scanner
// matches a record, it will call Keep() to make a safe copy.
func NewVolatileRecord(typ *TypeRecord, ts nano.Ts, raw zcode.Bytes) *Record {
	return &Record{
		Ts:          ts,
		Type:        typ,
		nonvolatile: false,
		Raw:         raw,
	}
}

// ZvalIter returns an iterator over the receiver's values.
func (r *Record) ZvalIter() zcode.Iter {
	return r.Raw.Iter()
}

// Width returns the number of columns in the record.
func (r *Record) Width() int { return len(r.Type.Columns) }

func (r *Record) Keep() *Record {
	if r.nonvolatile {
		return r
	}
	v := &Record{Ts: r.Ts, Type: r.Type, nonvolatile: true}
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
	return r.Type.HasField(field)
}

func (r *Record) Bytes() []byte {
	if r.Raw == nil {
		panic("this shouldn't happen")
	}
	return r.Raw
}

// TypeCheck checks that the value coding in Raw is structurally consistent
// with this value's descriptor.  It does not check that the actual leaf
// values when parsed are type compatible with the leaf types.
func (r *Record) TypeCheck() error {
	return checkRecord(r.Type, r.Raw)
}

func checkVector(typ *TypeArray, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	inner := InnerType(typ)
	it := zcode.Iter(body)
	for !it.Done() {
		body, container, err := it.Next()
		if err != nil {
			return err
		}
		switch v := inner.(type) {
		case *TypeRecord:
			if !container {
				return &RecordTypeError{Name: "<record element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkRecord(v, body); err != nil {
				return err
			}
		case *TypeArray:
			if !container {
				return &RecordTypeError{Name: "<array element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkVector(v, body); err != nil {
				return err
			}
		case *TypeSet:
			if !container {
				return &RecordTypeError{Name: "<set element>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkSet(v, body); err != nil {
				return err
			}
		case *TypeUnion:
			if !container {
				return &RecordTypeError{Name: "<union value>", Type: v.String(), Err: ErrNotContainer}
			}
			if err := checkUnion(v, body); err != nil {
				return err
			}
		default:
			if container {
				return &RecordTypeError{Name: "<array element>", Type: v.String(), Err: ErrNotPrimitive}
			}
		}
	}
	return nil
}

func checkUnion(typ *TypeUnion, body zcode.Bytes) error {
	if len(body) == 0 {
		return nil
	}
	it := zcode.Iter(body)
	v, container, err := it.Next()
	if err != nil {
		return err
	}
	if container {
		return ErrBadValue
	}
	index := zcode.DecodeCountedUvarint(v)
	inner, err := typ.TypeIndex(int(index))
	if err != nil {
		return err
	}
	body, container, err = it.Next()
	if err != nil {
		return err
	}
	switch v := inner.(type) {
	case *TypeRecord:
		if !container {
			return &RecordTypeError{Name: "<record element>", Type: v.String(), Err: ErrNotContainer}
		}
		if err := checkRecord(v, body); err != nil {
			return err
		}
	case *TypeArray:
		if !container {
			return &RecordTypeError{Name: "<array element>", Type: v.String(), Err: ErrNotContainer}
		}
		if err := checkVector(v, body); err != nil {
			return err
		}
	case *TypeSet:
		if !container {
			return &RecordTypeError{Name: "<set element>", Type: v.String(), Err: ErrNotContainer}
		}
		if err := checkSet(v, body); err != nil {
			return err
		}
	case *TypeUnion:
		if !container {
			return &RecordTypeError{Name: "<union value>", Type: v.String(), Err: ErrNotContainer}
		}
		if err := checkUnion(v, body); err != nil {
			return err
		}
	default:
		if container {
			return &RecordTypeError{Name: "<array element>", Type: v.String(), Err: ErrNotPrimitive}
		}
	}
	return nil
}

func checkSet(typ *TypeSet, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	inner := InnerType(typ)
	if IsContainerType(inner) {
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

func checkRecord(typ *TypeRecord, body zcode.Bytes) error {
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
		case *TypeRecord:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkRecord(v, body); err != nil {
				return err
			}
		case *TypeArray:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkVector(v, body); err != nil {
				return err
			}
		case *TypeSet:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkSet(v, body); err != nil {
				return err
			}
		case *TypeUnion:
			if !container {
				return &RecordTypeError{Name: col.Name, Type: col.Type.String(), Err: ErrNotContainer}
			}
			if err := checkUnion(v, body); err != nil {
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

// Value returns the indicated column as a Value.  If the column doesn't
// exist or another error occurs, the nil Value is returned.
func (r *Record) Value(col int) Value {
	zv, err := r.Slice(col)
	if err != nil {
		return Value{}
	}
	return Value{r.Type.Columns[col].Type, zv}
}

func (r *Record) ValueByField(field string) (Value, error) {
	col, ok := r.ColumnOfField(field)
	if !ok {
		return Value{}, ErrNoSuchField
	}
	return r.Value(col), nil
}

func (r *Record) ColumnOfField(field string) (int, bool) {
	return r.Type.ColumnOfField(field)
}

func (r *Record) TypeOfColumn(col int) Type {
	return r.Type.Columns[col].Type
}

func (r *Record) Access(field string) (Value, error) {
	col, ok := r.ColumnOfField(field)
	if !ok {
		return Value{}, ErrNoSuchField
	}
	return r.Value(col), nil
}

func (r *Record) AccessString(field string) (string, error) {
	v, err := r.Access(field)
	if err != nil {
		return "", err
	}
	switch v.Type.(type) {
	case *TypeOfString, *TypeOfBstring:
		return DecodeString(v.Bytes)
	default:
		return "", ErrTypeMismatch
	}
}

func (r *Record) AccessBool(field string) (bool, error) {
	v, err := r.Access(field)
	if err != nil {
		return false, err
	}
	if _, ok := v.Type.(*TypeOfBool); !ok {
		return false, ErrTypeMismatch
	}
	return DecodeBool(v.Bytes)
}

func (r *Record) AccessInt(field string) (int64, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	switch v.Type.(type) {
	case *TypeOfInt:
		return DecodeInt(v.Bytes)
	case *TypeOfCount:
		v, err := DecodeCount(v.Bytes)
		if v > math.MaxInt64 {
			return 0, errors.New("conversion from type count to int results in overflow")
		}
		return int64(v), err
	case *TypeOfPort:
		v, err := DecodePort(v.Bytes)
		return int64(v), err
	}
	return 0, ErrTypeMismatch
}

func (r *Record) AccessDouble(field string) (float64, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := v.Type.(*TypeOfDouble); !ok {
		return 0, ErrTypeMismatch
	}
	return DecodeDouble(v.Bytes)
}

func (r *Record) AccessIP(field string) (net.IP, error) {
	v, err := r.Access(field)
	if err != nil {
		return nil, err
	}
	if _, ok := v.Type.(*TypeOfAddr); !ok {
		return nil, ErrTypeMismatch
	}
	return DecodeAddr(v.Bytes)
}

func (r *Record) AccessTime(field string) (nano.Ts, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := v.Type.(*TypeOfTime); !ok {
		return 0, ErrTypeMismatch
	}
	return DecodeTime(v.Bytes)
}

func (r *Record) AccessTimeByColumn(colno int) (nano.Ts, error) {
	zv, err := r.Slice(colno)
	if err != nil {
		return 0, err
	}
	return DecodeTime(zv)
}

func (r *Record) String() string {
	return Value{r.Type, r.Raw}.String()
}

// MarshalJSON implements json.Marshaler.
func (r *Record) MarshalJSON() ([]byte, error) {
	return Value{r.Type, r.Raw}.MarshalJSON()
}
