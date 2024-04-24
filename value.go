package zed

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"unsafe"

	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

var (
	ErrMissingField = errors.New("record missing a field")
	ErrNotContainer = errors.New("expected container type, got primitive")
)

var (
	NullUint8    = Value{a: vTypePrimitiveNull | IDUint8}
	NullUint16   = Value{a: vTypePrimitiveNull | IDUint16}
	NullUint32   = Value{a: vTypePrimitiveNull | IDUint32}
	NullUint64   = Value{a: vTypePrimitiveNull | IDUint64}
	NullInt8     = Value{a: vTypePrimitiveNull | IDInt8}
	NullInt16    = Value{a: vTypePrimitiveNull | IDInt16}
	NullInt32    = Value{a: vTypePrimitiveNull | IDInt32}
	NullInt64    = Value{a: vTypePrimitiveNull | IDInt64}
	NullDuration = Value{a: vTypePrimitiveNull | IDDuration}
	NullTime     = Value{a: vTypePrimitiveNull | IDTime}
	NullFloat16  = Value{a: vTypePrimitiveNull | IDFloat16}
	NullFloat32  = Value{a: vTypePrimitiveNull | IDFloat32}
	NullFloat64  = Value{a: vTypePrimitiveNull | IDFloat64}
	NullBool     = Value{a: vTypePrimitiveNull | IDBool}
	NullBytes    = Value{a: vTypePrimitiveNull | IDBytes}
	NullString   = Value{a: vTypePrimitiveNull | IDString}
	NullIP       = Value{a: vTypePrimitiveNull | IDIP}
	NullNet      = Value{a: vTypePrimitiveNull | IDNet}
	NullType     = Value{a: vTypePrimitiveNull | IDType}
	Null         = Value{a: vTypePrimitiveNull | IDNull}

	False = NewBool(false)
	True  = NewBool(true)
)

const (
	vTypeUninitialized   = 0 << 60
	vTypeArena           = 1 << 60
	vTypePrimitive       = 2 << 60
	vTypePrimitiveNull   = 3 << 60
	vTypeBytes           = 4 << 60
	vTypeString          = 5 << 60
	vTypeMask            = uint64(0xf) << 60
	vLengthMask          = uint64(0x0f) << 56
	vPrimitiveTypeIDMask = 0xff

	arenaNullOffset = math.MaxUint32
	arenaUseValues  = uint64(1) << 63
)

func arenaDescSlot(d uint64) int    { return int(d & math.MaxUint32) }
func arenaDescTypeID(d uint64) int  { return int(d &^ arenaUseValues >> 32) }
func arenaDescValues(d uint64) bool { return d&arenaUseValues != 0 }

type Value struct {
	a uint64
	d uint64
}

func (v Value) Arena() (*Arena, bool) {
	if v.a&vTypeMask != vTypeArena {
		return nil, false
	}
	return (*Arena)(unsafe.Pointer(uintptr(v.a & ^vTypeMask))), true
}

func (v Value) arena() *Arena {
	arena, ok := v.Arena()
	if !ok {
		panic(v)
	}
	return arena
}

func (v Value) Ptr() *Value { return &v }

func (v Value) Type() Type {
	switch v.a & vTypeMask {
	case vTypeArena:
		return v.arena().type_(v.d)
	case vTypePrimitive, vTypePrimitiveNull:
		return idToType[v.a&vPrimitiveTypeIDMask]
	case vTypeBytes:
		return TypeBytes
	case vTypeString:
		return TypeString
	}
	panic(v)
}

var idToType = [...]Type{
	IDUint8:    TypeUint8,
	IDUint16:   TypeUint16,
	IDUint32:   TypeUint32,
	IDUint64:   TypeUint64,
	IDInt8:     TypeInt8,
	IDInt16:    TypeInt16,
	IDInt32:    TypeInt32,
	IDInt64:    TypeInt64,
	IDDuration: TypeDuration,
	IDTime:     TypeTime,
	IDFloat16:  TypeFloat16,
	IDFloat32:  TypeFloat32,
	IDFloat64:  TypeFloat64,
	IDBool:     TypeBool,
	IDBytes:    TypeBytes,
	IDString:   TypeString,
	IDIP:       TypeIP,
	IDNet:      TypeNet,
	IDType:     TypeType,
	IDNull:     TypeNull,
}

func NewUint(t Type, x uint64) Value    { return Value{uint64(vTypePrimitive | t.ID()), x} }
func NewUint8(u uint8) Value            { return NewUint(TypeUint8, uint64(u)) }
func NewUint16(u uint16) Value          { return NewUint(TypeUint16, uint64(u)) }
func NewUint32(u uint32) Value          { return NewUint(TypeUint32, uint64(u)) }
func NewUint64(u uint64) Value          { return NewUint(TypeUint64, u) }
func NewInt(t Type, x int64) Value      { return Value{uint64(vTypePrimitive | t.ID()), uint64(x)} }
func NewInt8(i int8) Value              { return NewInt(TypeInt8, int64(i)) }
func NewInt16(i int16) Value            { return NewInt(TypeInt16, int64(i)) }
func NewInt32(i int32) Value            { return NewInt(TypeInt32, int64(i)) }
func NewInt64(i int64) Value            { return NewInt(TypeInt64, i) }
func NewDuration(d nano.Duration) Value { return NewInt(TypeDuration, int64(d)) }
func NewTime(ts nano.Ts) Value          { return NewInt(TypeTime, int64(ts)) }
func NewFloat(t Type, x float64) Value {
	return Value{uint64(vTypePrimitive | t.ID()), uint64(math.Float64bits(x))}
}
func NewFloat16(f float32) Value { return NewFloat(TypeFloat16, float64(f)) }
func NewFloat32(f float32) Value { return NewFloat(TypeFloat32, float64(f)) }
func NewFloat64(f float64) Value { return NewFloat(TypeFloat64, f) }
func NewBool(x bool) Value       { return Value{vTypePrimitive | IDBool, boolToUint64(x)} }

func boolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func (v Value) typeID() int {
	switch v.a & vTypeMask {
	case vTypeArena:
		return v.arena().type_(v.d).ID()
	case vTypePrimitive, vTypePrimitiveNull:
		return int(v.a & vPrimitiveTypeIDMask)
	case vTypeBytes:
		return IDBytes
	case vTypeString:
		return IDString
	}
	panic(v)
}

// Uint returns v's underlying value.  It panics if v's underlying type is not
// TypeUint8, TypeUint16, TypeUint32, or TypeUint64.
func (v Value) Uint() uint64 {
	if !IsUnsigned(v.typeID()) {
		panic(fmt.Sprintf("zed.Value.Uint called on %T", v.Type()))
	}
	if v.a&vTypeMask == vTypePrimitive {
		return v.d
	}
	return DecodeUint(v.arena().bytes_(v.d))
}

// Int returns v's underlying value.  It panics if v's underlying type is not
// TypeInt8, TypeInt16, TypeInt32, TypeInt64, TypeDuration, or TypeTime.
func (v Value) Int() int64 {
	if !IsSigned(v.typeID()) {
		panic(fmt.Sprintf("zed.Value.Int called on %T", v.Type()))
	}
	if v.a&vTypeMask == vTypePrimitive {
		return int64(v.d)
	}
	return DecodeInt(v.arena().bytes_(v.d))
}

// Float returns v's underlying value.  It panics if v's underlying type is not
// TypeFloat16, TypeFloat32, or TypeFloat64.
func (v Value) Float() float64 {
	if !IsFloat(v.typeID()) {
		panic(fmt.Sprintf("zed.Value.Float called on %T", v.Type))
	}
	if v.a&vTypeMask == vTypePrimitive {
		return math.Float64frombits(v.d)
	}
	return DecodeFloat(v.arena().bytes_(v.d))
}

// Bool returns v's underlying value.  It panics if v's underlying type is not
// TypeBool.
func (v Value) Bool() bool {
	if v.typeID() != IDBool {
		panic(fmt.Sprintf("zed.Value.Bool called on %T", v.Type))
	}
	return v.asBool()
}

func (v Value) asBool() bool {
	if v.a&vTypeMask == vTypePrimitive {
		return v.d != 0
	}
	return DecodeBool(v.arena().bytes_(v.d))
}

// Bytes returns v's ZNG representation.
func (v Value) Bytes() zcode.Bytes {
	switch v.a & vTypeMask {
	case vTypeArena:
		return v.arena().bytes_(v.d)
	case vTypeBytes, vTypeString:
		var b [16]byte
		binary.BigEndian.PutUint64(b[:8], v.a)
		binary.BigEndian.PutUint64(b[8:], v.d)
		length := (v.a & vLengthMask) >> 56
		return b[1 : 1+length]
	case vTypePrimitive:
		switch v.a & vPrimitiveTypeIDMask {
		case IDUint8, IDUint16, IDUint32, IDUint64:
			return EncodeUint(v.d)
		case IDInt8, IDInt16, IDInt32, IDInt64, IDDuration, IDTime:
			return EncodeInt(int64(v.d))
		case IDFloat16:
			return EncodeFloat16(float32(math.Float64frombits(v.d)))
		case IDFloat32:
			return EncodeFloat32(float32(math.Float64frombits(v.d)))
		case IDFloat64:
			return EncodeFloat64(math.Float64frombits(v.d))
		case IDBool:
			return EncodeBool(v.d != 0)
		}
	case vTypePrimitiveNull:
		return nil
	}
	panic(v)
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
func (v Value) ArrayIndex(arena *Arena, idx int64) (Value, error) {
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
			return arena.New(vec.Type, bytes), nil
		}
	}
	return Null, ErrIndex
}

// Elements returns an array of Values for the given container type.
// Returns an error if the element is not an array or set.
func (v Value) Elements(arena *Arena) ([]Value, error) {
	innerType := InnerType(v.Type())
	if innerType == nil {
		return nil, ErrNotContainer
	}
	var elements []Value
	for it := v.Iter(); !it.Done(); {
		elements = append(elements, arena.New(innerType, it.Next()))
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

// IsNull returns true if and only if v is a null value of any type.
func (v Value) IsNull() bool {
	return v.a&vTypeMask == vTypePrimitiveNull ||
		v.a&vTypeMask == vTypeArena && v.arena().offsetsAndLengths[arenaDescSlot(v.d)]>>32 == arenaNullOffset
}

// Copy returns a copy of v that shares no storage.
func (v Value) Copy(arena *Arena) Value {
	a, ok := v.Arena()
	if !ok || a == arena {
		return v
	}
	if arenaDescValues(v.d) {
		slot := arenaDescSlot(v.d)
		start := a.offsetsAndLengths[slot] >> 32
		if start == arenaNullOffset {
			return arena.New(v.Type(), nil)
		}
		end := start + a.offsetsAndLengths[slot]&0xffffffff
		vals := make([]Value, 0, 32)
		for _, val := range a.values[start:end] {
			vals = append(vals, val.Copy(arena))
		}
		return arena.NewFromValues(v.Type(), vals)
	}
	return arena.New(v.Type(), v.Bytes())
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

func (v *Value) DerefByColumn(arena *Arena, col int) *Value {
	if v != nil {
		if bytes := v.nth(col); bytes != nil {
			return arena.New(v.Fields()[col].Type, bytes).Ptr()
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

func (v *Value) Deref(arena *Arena, field string) *Value {
	if v == nil {
		return nil
	}
	i, ok := v.IndexOfField(field)
	if !ok {
		return nil
	}
	return v.DerefByColumn(arena, i)
}

func (v *Value) DerefPath(arena *Arena, path field.Path) *Value {
	for len(path) != 0 {
		v = v.Deref(arena, path[0])
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
	if v == nil || v.typeID() != IDBool {
		return false
	}
	return v.asBool()
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
func (v Value) Under(arena *Arena) Value {
	switch v.Type().(type) {
	case *TypeUnion, *TypeNamed:
		return v.under(arena)
	}
	// This is the common case; make sure the compiler can inline it.
	return v
}

// under contains logic for Under that the compiler won't inline.
func (v Value) under(arena *Arena) Value {
	typ, bytes := v.Type(), v.Bytes()
	for {
		typ = TypeUnder(typ)
		union, ok := typ.(*TypeUnion)
		if !ok {
			return arena.New(typ, bytes)
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
