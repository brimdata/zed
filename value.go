package zed

import (
	"bytes"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/exp/slices"
)

var (
	ErrMissingField  = errors.New("record missing a field")
	ErrExtraField    = errors.New("record with extra field")
	ErrNotContainer  = errors.New("expected container type, got primitive")
	ErrNotPrimitive  = errors.New("expected primitive type, got container")
	ErrTypeIDInvalid = errors.New("zng type ID out of range")
	ErrBadValue      = errors.New("malformed zng value")
	ErrBadFormat     = errors.New("malformed zng record")
	ErrTypeMismatch  = errors.New("type/value mismatch")
	ErrTypeSyntax    = errors.New("syntax error parsing type string")
)

var (
	NullUint8    = &Value{Type: TypeUint8}
	NullUint16   = &Value{Type: TypeUint16}
	NullUint32   = &Value{Type: TypeUint32}
	NullUint64   = &Value{Type: TypeUint64}
	NullInt8     = &Value{Type: TypeInt8}
	NullInt16    = &Value{Type: TypeInt16}
	NullInt32    = &Value{Type: TypeInt32}
	NullInt64    = &Value{Type: TypeInt64}
	NullDuration = &Value{Type: TypeDuration}
	NullTime     = &Value{Type: TypeTime}
	NullFloat16  = &Value{Type: TypeFloat16}
	NullFloat32  = &Value{Type: TypeFloat32}
	NullFloat64  = &Value{Type: TypeFloat64}
	NullBool     = &Value{Type: TypeBool}
	NullBytes    = &Value{Type: TypeBytes}
	NullString   = &Value{Type: TypeString}
	NullIP       = &Value{Type: TypeIP}
	NullNet      = &Value{Type: TypeNet}
	NullType     = &Value{Type: TypeType}
	Null         = &Value{Type: TypeNull}
)

type Allocator interface {
	NewValue(Type, zcode.Bytes) *Value
	CopyValue(*Value) *Value
}

type Value struct {
	Type  Type
	Bytes zcode.Bytes
}

func NewValue(zt Type, zb zcode.Bytes) *Value {
	return &Value{zt, zb}
}

func (v *Value) IsContainer() bool {
	return IsContainerType(v.Type)
}

// String implements fmt.Stringer.String.  It should only be used for logs,
// debugging, etc.  Any caller that requires a specific output format should use
// FormatAs() instead.
func (v *Value) String() string {
	return fmt.Sprintf("%s: %s", v.Type, v.Encode(nil))
}

// Encode appends the ZNG representation of this value to the passed in
// argument and returns the resulting zcode.Bytes (which may or may not
// be the same underlying buffer, as with append(), depending on its capacity)
func (v *Value) Encode(dst zcode.Bytes) zcode.Bytes {
	//XXX don't need this...
	return zcode.Append(dst, v.Bytes)
}

func (v *Value) Iter() zcode.Iter {
	return v.Bytes.Iter()
}

// If the passed-in element is an array, attempt to get the idx'th
// element, and return its type and raw representation.  Returns an
// error if the passed-in element is not an array or if idx is
// outside the array bounds.
func (v *Value) ArrayIndex(idx int64) (Value, error) {
	vec, ok := v.Type.(*TypeArray)
	if !ok {
		return Value{}, ErrNotArray
	}
	if idx < 0 {
		return Value{}, ErrIndex
	}
	for i, it := 0, v.Iter(); !it.Done(); i++ {
		bytes := it.Next()
		if i == int(idx) {
			return Value{vec.Type, bytes}, nil
		}
	}
	return Value{}, ErrIndex
}

// Elements returns an array of Values for the given container type.
// Returns an error if the element is not an array or set.
func (v *Value) Elements() ([]Value, error) {
	innerType := InnerType(v.Type)
	if innerType == nil {
		return nil, ErrNotContainer
	}
	var elements []Value
	for it := v.Iter(); !it.Done(); {
		elements = append(elements, Value{innerType, it.Next()})
	}
	return elements, nil
}

func (v *Value) ContainerLength() (int, error) {
	switch v.Type.(type) {
	case *TypeSet, *TypeArray:
		if v.IsNull() {
			return 0, nil
		}
		var n int
		for it := v.Iter(); !it.Done(); {
			it.Next()
			n++
		}
		return n, nil
	case *TypeMap:
		if v.Bytes == nil {
			return 0, nil
		}
		var n int
		for it := v.Iter(); !it.Done(); {
			it.Next()
			it.Next()
			n++
		}
		return n, nil
	default:
		return -1, ErrNotContainer
	}
}

// IsNull returns true if and only if v is a null value of any type.
func (v *Value) IsNull() bool {
	return v.Bytes == nil
}

// Copy returns a copy of v that does not share v.Bytes.  The copy's Bytes field
// is nil if and only if v.Bytes is nil.
func (v *Value) Copy() *Value {
	return &Value{v.Type, slices.Clone(v.Bytes)}
}

// CopyFrom copies from into v, reusing v.Bytes if possible and setting v.Bytes
// to nil if and only if from.Bytes is nil.
func (v *Value) CopyFrom(from *Value) {
	v.Type = from.Type
	if from.Bytes == nil {
		v.Bytes = nil
	} else if v.Bytes == nil {
		v.Bytes = slices.Clone(from.Bytes)
	} else {
		v.Bytes = append(v.Bytes[:0], from.Bytes...)
	}
}

func (v *Value) IsString() bool {
	_, ok := TypeUnder(v.Type).(*TypeOfString)
	return ok
}

func (v *Value) IsError() bool {
	_, ok := TypeUnder(v.Type).(*TypeError)
	return ok
}

func (v *Value) IsMissing() bool {
	if v == nil {
		return true
	}
	if typ, ok := v.Type.(*TypeError); ok {
		return typ.IsMissing(v.Bytes)
	}
	return false
}

func (v *Value) IsQuiet() bool {
	if typ, ok := v.Type.(*TypeError); ok {
		return typ.IsQuiet(v.Bytes)
	}
	return false
}

func (v *Value) Equal(p Value) bool {
	return v.Type == p.Type && bytes.Equal(v.Bytes, p.Bytes)
}

func (r *Value) HasField(field string) bool {
	return TypeRecordOf(r.Type).HasField(field)
}

// Walk traverses a value in depth-first order, calling a
// Visitor on the way.
func (r *Value) Walk(rv Visitor) error {
	return Walk(r.Type, r.Bytes, rv)
}

func (r *Value) nth(column int) zcode.Bytes {
	var zv zcode.Bytes
	for i, it := 0, r.Bytes.Iter(); i <= column; i++ {
		if it.Done() {
			return nil
		}
		zv = it.Next()
	}
	return zv
}

func (r *Value) Columns() []Column {
	return TypeRecordOf(r.Type).Columns
}

func (v *Value) DerefByColumn(col int) *Value {
	if v != nil {
		if bytes := v.nth(col); bytes != nil {
			v = &Value{v.Columns()[col].Type, bytes}
		} else {
			v = nil
		}
	}
	return v
}

func (v *Value) ColumnOfField(field string) (int, bool) {
	if typ := TypeRecordOf(v.Type); typ != nil {
		return typ.ColumnOfField(field)
	}
	return 0, false
}

func (v *Value) Deref(field string) *Value {
	if v == nil {
		return nil
	}
	col, ok := v.ColumnOfField(field)
	if !ok {
		return nil
	}
	return v.DerefByColumn(col)
}

func (v *Value) DerefPath(path field.Path) *Value {
	for len(path) != 0 {
		v = v.Deref(path[0])
		path = path[1:]
	}
	return v
}

func (v *Value) AsString() string {
	if v != nil && TypeUnder(v.Type) == TypeString {
		return DecodeString(v.Bytes)
	}
	return ""
}

func (v *Value) AsBool() bool {
	if v != nil && TypeUnder(v.Type) == TypeBool {
		return DecodeBool(v.Bytes)
	}
	return false
}

func (v *Value) AsInt() int64 {
	if v != nil {
		switch TypeUnder(v.Type).(type) {
		case *TypeOfUint8, *TypeOfUint16, *TypeOfUint32, *TypeOfUint64:
			return int64(DecodeUint(v.Bytes))
		case *TypeOfInt8, *TypeOfInt16, *TypeOfInt32, *TypeOfInt64:
			return DecodeInt(v.Bytes)
		}
	}
	return 0
}

func (v *Value) AsTime() nano.Ts {
	if v != nil && TypeUnder(v.Type) == TypeTime {
		return DecodeTime(v.Bytes)
	}
	return 0
}

func (v *Value) MissingAsNull() *Value {
	if v.IsMissing() {
		return Null
	}
	return v
}

func (v *Value) Under() *Value {
	typ := v.Type
	if _, ok := typ.(*TypeNamed); !ok {
		if _, ok := typ.(*TypeUnion); !ok {
			// common fast path
			return v
		}
	}
	bytes := v.Bytes
	for {
		typ = TypeUnder(typ)
		union, ok := typ.(*TypeUnion)
		if !ok {
			return NewValue(typ, bytes)
		}
		typ, bytes = union.Untag(bytes)
	}
}

// Validate checks that v.Bytes is structurally consistent
// with v.Type.  It does not check that the actual leaf
// values when parsed are type compatible with the leaf types.
func (v *Value) Validate() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %+v\n%s", r, debug.Stack())
		}
	}()
	return v.Walk(func(typ Type, body zcode.Bytes) error {
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
		tagAndBody := it.NextTagAndBody()
		if prev != nil {
			switch bytes.Compare(prev, tagAndBody) {
			case 0:
				return errors.New("invalid ZNG: duplicate set element")
			case 1:
				return errors.New("invalid ZNG: set elements not sorted")
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
	if selector := DecodeUint(body); int(selector) >= len(typ.Symbols) {
		return errors.New("enum selector out of range")
	}
	return nil
}
