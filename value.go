package zed

import (
	"bytes"
	"errors"
	"fmt"
	"net/netip"
	"runtime/debug"

	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

var (
	ErrMissingField = errors.New("record missing a field")
	ErrNotContainer = errors.New("expected container type, got primitive")
)

var (
	NullUint8    = newNullValue(IDUint8)
	NullUint16   = newNullValue(IDUint16)
	NullUint32   = newNullValue(IDUint32)
	NullUint64   = newNullValue(IDUint64)
	NullInt8     = newNullValue(IDInt8)
	NullInt16    = newNullValue(IDInt16)
	NullInt32    = newNullValue(IDInt32)
	NullInt64    = newNullValue(IDInt64)
	NullDuration = newNullValue(IDDuration)
	NullTime     = newNullValue(IDTime)
	NullFloat16  = newNullValue(IDFloat16)
	NullFloat32  = newNullValue(IDFloat32)
	NullFloat64  = newNullValue(IDFloat64)
	NullBool     = newNullValue(IDBool)
	NullBytes    = newNullValue(IDBytes)
	NullString   = newNullValue(IDString)
	NullIP       = newNullValue(IDIP)
	NullNet      = newNullValue(IDNet)
	NullType     = newNullValue(IDType)
	Null         = newNullValue(IDNull)

	False = NewBool(false)
	True  = NewBool(true)
)

func newNullValue(id int) Value { return Value{vPrimitiveNull | uint64(id), 0} }

type Allocator interface{}

func (v Value) Ptr() *Value { return &v }

func NewUint8(u uint8) Value                 { return NewUint(TypeUint8, uint64(u)) }
func NewUint16(u uint16) Value               { return NewUint(TypeUint16, uint64(u)) }
func NewUint32(u uint32) Value               { return NewUint(TypeUint32, uint64(u)) }
func NewUint64(u uint64) Value               { return NewUint(TypeUint64, u) }
func NewInt8(i int8) Value                   { return NewInt(TypeInt8, int64(i)) }
func NewInt16(i int16) Value                 { return NewInt(TypeInt16, int64(i)) }
func NewInt32(i int32) Value                 { return NewInt(TypeInt32, int64(i)) }
func NewInt64(i int64) Value                 { return NewInt(TypeInt64, i) }
func NewDuration(d nano.Duration) Value      { return NewInt(TypeDuration, int64(d)) }
func NewTime(ts nano.Ts) Value               { return NewInt(TypeTime, int64(ts)) }
func NewFloat16(f float32) Value             { return NewFloat(TypeFloat16, float64(f)) }
func NewFloat32(f float32) Value             { return NewFloat(TypeFloat32, float64(f)) }
func NewFloat64(f float64) Value             { return NewFloat(TypeFloat64, f) }
func (a *Arena) NewIP(x netip.Addr) Value    { return a.NewFromBytes(TypeIP, EncodeIP(x)) }
func (a *Arena) NewNet(p netip.Prefix) Value { return a.NewFromBytes(TypeNet, EncodeNet(p)) }
func (a *Arena) NewTypeValue(t Type) Value {
	return a.NewFromBytes(TypeNet, EncodeTypeValue(t))
}

func boolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func (v Value) IsContainer() bool {
	return IsContainerType(v.Type())
}

// String implements fmt.Stringer.String.  It should only be used for logs,
// debugging, etc.  Any caller that requires a specific output format should use
// FormatAs() instead.
func (v Value) String() string {
	return fmt.Sprintf("%s: %s", v.Type(), v.Encode(nil))
}

// Encode appends the ZNG representation of this value to the passed in
// argument and returns the resulting zcode.Bytes (which may or may not
// be the same underlying buffer, as with append(), depending on its capacity)
func (v Value) Encode(dst zcode.Bytes) zcode.Bytes {
	//XXX don't need this...
	return zcode.Append(dst, v.Bytes())
}

func (v Value) Iter() zcode.Iter {
	return v.Bytes().Iter()
}

// If the passed-in element is an array, attempt to get the idx'th
// element, and return its type and raw representation.  Returns an
// error if the passed-in element is not an array or if idx is
// outside the array bounds.
func (v Value) ArrayIndex(idx int64) (Value, error) {
	vec, ok := v.Type().(*TypeArray)
	if !ok {
		return Null, ErrNotArray
	}
	if idx < 0 {
		return Null, ErrIndex
	}
	for i, it := 0, v.Iter(); !it.Done(); i++ {
		bytes := it.Next()
		if i == int(idx) {
			return v.arena().NewValue(vec.Type, bytes), nil
		}
	}
	return Null, ErrIndex
}

// Elements returns an array of Values for the given container type.
// Returns an error if the element is not an array or set.
func (v Value) Elements() ([]Value, error) {
	innerType := InnerType(v.Type())
	if innerType == nil {
		return nil, ErrNotContainer
	}
	var elements []Value
	for it := v.Iter(); !it.Done(); {
		elements = append(elements, v.arena().NewValue(innerType, it.Next()))
	}
	return elements, nil
}

func (v Value) ContainerLength() (int, error) {
	switch v.Type().(type) {
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
		if v.IsNull() {
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

func (v Value) Copy() Value {
	return v
}

// CopyFrom copies from into v, reusing v's storage if possible.
func (v *Value) CopyFrom(from Value) {
	*v = from
}

func (v Value) IsString() bool {
	_, ok := TypeUnder(v.Type()).(*TypeOfString)
	return ok
}

func (v Value) IsError() bool {
	_, ok := TypeUnder(v.Type()).(*TypeError)
	return ok
}

func (v *Value) IsMissing() bool {
	if v == nil {
		return true
	}
	if typ, ok := v.Type().(*TypeError); ok {
		return typ.IsMissing(v.Bytes())
	}
	return false
}

func (v Value) IsQuiet() bool {
	if typ, ok := v.Type().(*TypeError); ok {
		return typ.IsQuiet(v.Bytes())
	}
	return false
}

// Equal reports whether p and v have the same type and the same ZNG
// representation.
func (v Value) Equal(p Value) bool {
	if v == p {
		return true
	}
	if v.Type() != p.Type() {
		return false
	}
	return bytes.Equal(v.Bytes(), p.Bytes())
}

func (r Value) HasField(field string) bool {
	return TypeRecordOf(r.Type()).HasField(field)
}

// Walk traverses a value in depth-first order, calling a
// Visitor on the way.
func (r Value) Walk(rv Visitor) error {
	return Walk(r.Type(), r.Bytes(), rv)
}

func (r Value) nth(n int) zcode.Bytes {
	var zv zcode.Bytes
	for i, it := 0, r.Bytes().Iter(); i <= n; i++ {
		if it.Done() {
			return nil
		}
		zv = it.Next()
	}
	return zv
}

func (r Value) Fields() []Field {
	return TypeRecordOf(r.Type()).Fields
}

func (v *Value) DerefByColumn(col int) *Value {
	if v != nil {
		if bytes := v.nth(col); bytes != nil {
			return v.arena().NewValue(v.Fields()[col].Type, bytes).Ptr()
		}
	}
	return nil
}

func (v Value) IndexOfField(field string) (int, bool) {
	if typ := TypeRecordOf(v.Type()); typ != nil {
		return typ.IndexOfField(field)
	}
	return 0, false
}

func (v *Value) Deref(field string) *Value {
	if v == nil {
		return nil
	}
	i, ok := v.IndexOfField(field)
	if !ok {
		return nil
	}
	return v.DerefByColumn(i)
}

func (v *Value) DerefPath(path field.Path) *Value {
	for len(path) != 0 {
		v = v.Deref(path[0])
		path = path[1:]
	}
	return v
}

func (v *Value) AsString() string {
	if v != nil && TypeUnder(v.Type()) == TypeString {
		return DecodeString(v.Bytes())
	}
	return ""
}

// AsBool returns v's underlying value.  It returns false if v is nil or v's
// underlying type is not TypeBool.
func (v *Value) AsBool() bool {
	if v != nil && TypeUnder(v.Type()) == TypeBool {
		return v.Bool()
	}
	return false
}

func (v *Value) AsInt() int64 {
	if v != nil {
		switch TypeUnder(v.Type()).(type) {
		case *TypeOfUint8, *TypeOfUint16, *TypeOfUint32, *TypeOfUint64:
			return int64(v.Uint())
		case *TypeOfInt8, *TypeOfInt16, *TypeOfInt32, *TypeOfInt64:
			return v.Int()
		}
	}
	return 0
}

func (v *Value) AsTime() nano.Ts {
	if v != nil && TypeUnder(v.Type()) == TypeTime {
		return DecodeTime(v.Bytes())
	}
	return 0
}

func (v *Value) MissingAsNull() Value {
	if v.IsMissing() {
		return Null
	}
	return *v
}

// Under resolves named types and untags unions repeatedly, returning a value
// guaranteed to have neither a named type nor a union type.
func (v Value) Under() Value {
	switch v.Type().(type) {
	case *TypeUnion, *TypeNamed:
		return v.under()
	}
	// This is the common case; make sure the compiler can inline it.
	return v
}

// under contains logic for Under that the compiler won't inline.
func (v Value) under() Value {
	typ, bytes := v.Type(), v.Bytes()
	for {
		typ = TypeUnder(typ)
		union, ok := typ.(*TypeUnion)
		if !ok {
			return v.arena().NewValue(typ, bytes)
		}
		typ, bytes = union.Untag(bytes)
	}
}

// Validate checks that v.Bytes is structurally consistent
// with v.Type.  It does not check that the actual leaf
// values when parsed are type compatible with the leaf types.
func (v Value) Validate() (err error) {
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
