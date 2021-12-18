package zed

import (
	"bytes"
	"errors"
	"math"
	"net"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

var (
	ErrMissingField   = errors.New("record missing a field")
	ErrExtraField     = errors.New("record with extra field")
	ErrNotContainer   = errors.New("expected container type, got primitive")
	ErrNotPrimitive   = errors.New("expected primitive type, got container")
	ErrTypeIDExists   = errors.New("zng type ID exists")
	ErrTypeIDInvalid  = errors.New("zng type ID out of range")
	ErrBadValue       = errors.New("malformed zng value")
	ErrBadFormat      = errors.New("malformed zng record")
	ErrTypeMismatch   = errors.New("type/value mismatch")
	ErrColumnMismatch = errors.New("zng record mismatch between columns in type and columns in value")
	ErrCorruptColumns = errors.New("wrong number of columns in zng record value")
)

type RecordTypeError struct {
	Name string
	Type string
	Err  error
}

func (r *RecordTypeError) Error() string { return r.Name + " (" + r.Type + "): " + r.Err.Error() }
func (r *RecordTypeError) Unwrap() error { return r.Err }

// FieldIter returns a fieldIter iterator over the receiver's values.
func (r *Value) FieldIter() fieldIter {
	return fieldIter{
		stack: []iterInfo{{
			iter: r.Bytes.Iter(),
			typ:  TypeRecordOf(r.Type),
		}},
	}
}

func (r *Value) HasField(field string) bool {
	return TypeRecordOf(r.Type).HasField(field)
}

// Walk traverses a value in depth-first order, calling a
// Visitor on the way.
func (r *Value) Walk(rv Visitor) error {
	return Walk(r.Type, r.Bytes, rv)
}

// TypeCheck checks that the Bytes field is structurally consistent
// with this value's Type.  It does not check that the actual leaf
// values when parsed are type compatible with the leaf types.
func (r *Value) TypeCheck() error {
	return r.Walk(func(typ Type, body zcode.Bytes) error {
		if typset, ok := typ.(*TypeSet); ok {
			if err := checkSet(typset, body); err != nil {
				return err
			}
			return SkipContainer
		}
		if typ, ok := typ.(*TypeEnum); ok {
			if err := checkEnum(typ, body); err != nil {
				return err
			}
			return SkipContainer
		}
		return nil
	})
}

func checkSet(typ *TypeSet, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	it := body.Iter()
	var prev zcode.Bytes
	for !it.Done() {
		tagAndBody, _, err := it.NextTagAndBody()
		if err != nil {
			return err
		}
		if prev != nil {
			switch bytes.Compare(prev, tagAndBody) {
			case 0:
				err := errors.New("duplicate element")
				return &RecordTypeError{Name: "<set element>", Type: typ.String(), Err: err}
			case 1:
				err := errors.New("elements not sorted")
				return &RecordTypeError{Name: "<set element>", Type: typ.String(), Err: err}
			}
		}
		prev = tagAndBody
	}
	return nil
}

func checkEnum(typ *TypeEnum, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	selector, err := DecodeUint(body)
	if err != nil {
		return err
	}
	if int(selector) >= len(typ.Symbols) {
		return errors.New("enum selector out of range")
	}
	return nil
}

// Slice returns the encoded zcode.Bytes corresponding to the indicated
// column or an error if a problem was encountered.
func (r *Value) Slice(column int) (zcode.Bytes, error) {
	var zv zcode.Bytes
	for i, it := 0, r.Bytes.Iter(); i <= column; i++ {
		if it.Done() {
			return nil, ErrMissing
		}
		var err error
		zv, _, err = it.Next()
		if err != nil {
			return nil, err
		}
	}
	return zv, nil
}

func (r *Value) Columns() []Column {
	return TypeRecordOf(r.Type).Columns
}

// Value returns the indicated column as a Value.  If the column doesn't
// exist or another error occurs, the nil Value is returned.
func (r *Value) ValueByColumn(col int) Value {
	zv, err := r.Slice(col)
	if err != nil {
		return Value{}
	}
	return Value{r.Columns()[col].Type, zv}
}

func (r *Value) ValueByField(field string) (Value, error) {
	col, ok := r.ColumnOfField(field)
	if !ok {
		return Value{}, ErrMissing
	}
	return r.ValueByColumn(col), nil
}

func (r *Value) ColumnOfField(field string) (int, bool) {
	return TypeRecordOf(r.Type).ColumnOfField(field)
}

func (r *Value) TypeOfColumn(col int) Type {
	return TypeRecordOf(r.Type).Columns[col].Type
}

func (r *Value) Access(field string) (Value, error) {
	col, ok := r.ColumnOfField(field)
	if !ok {
		return Value{}, ErrMissing
	}
	return r.ValueByColumn(col), nil
}

func (r *Value) Deref(path field.Path) (Value, error) {
	v := *r
	for _, f := range path {
		typ := TypeRecordOf(v.Type)
		if typ == nil {
			return Value{}, errors.New("field access on non-record value")
		}
		var err error
		v, err = NewValue(typ, v.Bytes).Access(f)
		if err != nil {
			return Value{}, err
		}
	}
	return v, nil
}

func (r *Value) AccessString(field string) (string, error) {
	v, err := r.Access(field)
	if err != nil {
		return "", err
	}
	switch AliasOf(v.Type).(type) {
	case *TypeOfString, *TypeOfBstring:
		return DecodeString(v.Bytes)
	default:
		return "", ErrTypeMismatch
	}
}

func (r *Value) AccessBool(field string) (bool, error) {
	v, err := r.Access(field)
	if err != nil {
		return false, err
	}
	if _, ok := AliasOf(v.Type).(*TypeOfBool); !ok {
		return false, ErrTypeMismatch
	}
	return DecodeBool(v.Bytes)
}

func (r *Value) AccessInt(field string) (int64, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	switch AliasOf(v.Type).(type) {
	case *TypeOfUint8:
		b, err := DecodeUint(v.Bytes)
		return int64(b), err
	case *TypeOfInt16, *TypeOfInt32, *TypeOfInt64:
		return DecodeInt(v.Bytes)
	case *TypeOfUint16, *TypeOfUint32:
		v, err := DecodeUint(v.Bytes)
		return int64(v), err
	case *TypeOfUint64:
		v, err := DecodeUint(v.Bytes)
		if v > math.MaxInt64 {
			return 0, errors.New("conversion from uint64 to signed int results in overflow")
		}
		return int64(v), err
	}
	return 0, ErrTypeMismatch
}

func (r *Value) AccessIP(field string) (net.IP, error) {
	v, err := r.Access(field)
	if err != nil {
		return nil, err
	}
	if _, ok := AliasOf(v.Type).(*TypeOfIP); !ok {
		return nil, ErrTypeMismatch
	}
	return DecodeIP(v.Bytes)
}

func (r *Value) AccessTime(field string) (nano.Ts, error) {
	v, err := r.Access(field)
	if err != nil {
		return 0, err
	}
	if _, ok := AliasOf(v.Type).(*TypeOfTime); !ok {
		return 0, ErrTypeMismatch
	}
	return DecodeTime(v.Bytes)
}

func (r *Value) AccessTimeByColumn(colno int) (nano.Ts, error) {
	zv, err := r.Slice(colno)
	if err != nil {
		return 0, err
	}
	return DecodeTime(zv)
}
