package zed21

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/brimdata/zed/zcode"
)

var ErrTypeSyntax = errors.New("syntax error parsing type string")

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

func (v *Value) IsContainer() bool {
	return IsContainerType(v.Type)
}

func badZNG(err error, t Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

func (v *Value) Iter() zcode.Iter {
	return v.Bytes.Iter()
}

// IsNull returns true if and only if v is a null value of any type.
func (v *Value) IsNull() bool {
	return v.Bytes == nil
}

// Copy returns a copy of v that does not share v.Bytes.  The copy's Bytes field
// is nil if and only if v.Bytes is nil.
func (v *Value) Copy() *Value {
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

func (v *Value) Equal(p Value) bool {
	return v.Type == p.Type && bytes.Equal(v.Bytes, p.Bytes)
}
