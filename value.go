package zed

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/brimdata/zed/zcode"
)

var (
	ErrNotNumber  = errors.New("not a number")
	ErrTypeSyntax = errors.New("syntax error parsing type string")
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
	CopyValue(Value) *Value
}

type Value struct {
	Type  Type
	Bytes zcode.Bytes
}

func NewValue(zt Type, zb zcode.Bytes) *Value {
	return &Value{zt, zb}
}

func (v Value) IsContainer() bool {
	return IsContainerType(v.Type)
}

func (v Value) MarshalJSON() ([]byte, error) {
	if v.Bytes == nil {
		return json.Marshal(nil)
	}
	object, err := v.Type.Marshal(v.Bytes)
	if err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

func badZNG(err error, t Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

// String implements fmt.Stringer.String.  It should only be used for logs,
// debugging, etc.  Any caller that requires a specific output format should use
// FormatAs() instead.
func (v Value) String() string {
	return fmt.Sprintf("%s: %s", v.Type, v.Encode(nil))
}

// Encode appends the ZNG representation of this value to the passed in
// argument and returns the resulting zcode.Bytes (which may or may not
// be the same underlying buffer, as with append(), depending on its capacity)
func (v Value) Encode(dst zcode.Bytes) zcode.Bytes {
	if IsContainerType(v.Type) {
		return zcode.AppendContainer(dst, v.Bytes)
	}
	return zcode.AppendPrimitive(dst, v.Bytes)
}

func (v Value) Iter() zcode.Iter {
	return v.Bytes.Iter()
}

// If the passed-in element is an array, attempt to get the idx'th
// element, and return its type and raw representation.  Returns an
// error if the passed-in element is not an array or if idx is
// outside the array bounds.
func (v Value) ArrayIndex(idx int64) (Value, error) {
	vec, ok := v.Type.(*TypeArray)
	if !ok {
		return Value{}, ErrNotArray
	}
	if idx < 0 {
		return Value{}, ErrIndex
	}
	for i, it := 0, v.Iter(); !it.Done(); i++ {
		zv, _, err := it.Next()
		if err != nil {
			return Value{}, err
		}
		if i == int(idx) {
			return Value{vec.Type, zv}, nil
		}
	}
	return Value{}, ErrIndex
}

// Elements returns an array of Values for the given container type.
// Returns an error if the element is not an array or set.
func (v Value) Elements() ([]Value, error) {
	innerType := InnerType(v.Type)
	if innerType == nil {
		return nil, ErrNotContainer
	}
	var elements []Value
	for it := v.Iter(); !it.Done(); {
		zv, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		elements = append(elements, Value{innerType, zv})
	}
	return elements, nil
}

func (v Value) ContainerLength() (int, error) {
	switch v.Type.(type) {
	case *TypeSet, *TypeArray:
		if v.IsNull() {
			return 0, nil
		}
		var n int
		for it := v.Iter(); !it.Done(); {
			if _, _, err := it.Next(); err != nil {
				return -1, err
			}
			n++
		}
		return n, nil
	case *TypeMap:
		if v.Bytes == nil {
			return 0, nil
		}
		var n int
		for it := v.Iter(); !it.Done(); {
			if _, _, err := it.Next(); err != nil {
				return -1, err
			}
			if _, _, err := it.Next(); err != nil {
				return -1, err
			}
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
func (v Value) Copy() *Value {
	var b zcode.Bytes
	if v.Bytes != nil {
		b = make(zcode.Bytes, len(v.Bytes))
		copy(b, v.Bytes)
	}
	return &Value{v.Type, b}
}

// CopyFrom copies from into v, reusing v.Bytes if possible and setting v.Bytes
// to nil if and only if from.Bytes is nil.
func (v *Value) CopyFrom(from *Value) {
	v.Type = from.Type
	if from.Bytes == nil {
		v.Bytes = nil
	} else if v.Bytes == nil {
		v.Bytes = make(zcode.Bytes, len(from.Bytes))
		copy(v.Bytes, from.Bytes)
	} else {
		v.Bytes = append(v.Bytes[:0], from.Bytes...)
	}
}

func (v Value) IsStringy() bool {
	return IsStringy(v.Type.ID())
}

func (v Value) IsError() bool {
	return AliasOf(v.Type) == TypeError
}

func (v Value) IsMissing() bool {
	return v.Type == Missing.Type && bytes.Equal(v.Bytes, Missing.Bytes)
}

func (v Value) IsQuiet() bool {
	return v.Type == Quiet.Type && bytes.Equal(v.Bytes, Quiet.Bytes)
}

func (v Value) Equal(p Value) bool {
	return v.Type == p.Type && bytes.Equal(v.Bytes, p.Bytes)
}
