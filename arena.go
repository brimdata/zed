package zed

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/brimdata/zed/zcode"
)

type Arena struct {
	zctx *Context
	pool *sync.Pool
	refs int32

	offsets []uint32
	lengths []uint32
	bytes   []byte
	values  []Value
}

func NewArena(zctx *Context) *Arena { return NewArenaInPool(zctx, nil) }
func NewArenaInPool(zctx *Context, pool *sync.Pool) *Arena {
	if zctx == nil {
		panic("zctx==nil in NewArenaInPool")
	}
	return &Arena{zctx: zctx, pool: pool, refs: 1}
}

func (a *Arena) AddRefTo(v interface{ Ref() }) { v.Ref() }

func (a *Arena) Ref() { atomic.AddInt32(&a.refs, 1) }

func (a *Arena) Unref() {
	if refs := atomic.AddInt32(&a.refs, -1); refs == 0 {
		if a.pool != nil {
			a.pool.Put(a)
		}
	} else if refs < 0 {
		panic("negative arena reference count")
	}
}

func (a *Arena) Zctx() *Context { return a.zctx }

func (a *Arena) Grow(n int) { a.bytes = slices.Grow(a.bytes, n) }

func (a *Arena) Reset() {
	if true {
		clear(a.offsets)
		clear(a.lengths)
		clear(a.bytes)
		clear(a.values)
	}
	a.offsets = a.offsets[:0]
	a.lengths = a.lengths[:0]
	a.bytes = a.bytes[:0]
	a.values = a.values[:0]
}

const arenaNullOffset = math.MaxUint32
const arenaUseValues = uint64(1) << 63

func arenaDescSlot(d uint64) int    { return int(d & math.MaxUint32) }
func arenaDescTypeID(d uint64) int  { return int(d >> 32 &^ arenaUseValues) }
func arenaDescValues(d uint64) bool { return d&arenaUseValues != 0 }

func (a *Arena) NewValue(t Type, b zcode.Bytes) Value { return a.NewFromBytes(t, b) }

func (a *Arena) NewFromBytes(t Type, b zcode.Bytes) Value {
	if len(a.bytes) > math.MaxUint32 {
		panic("offset overflow")
	}
	if len(b) > math.MaxUint32 {
		panic("length overflow")
	}
	off := uint32(len(a.bytes))
	if b == nil {
		off = arenaNullOffset
	}
	a.offsets = append(a.offsets, off)
	a.lengths = append(a.lengths, uint32(len(b)))
	if len(a.bytes) == 0 {
		a.bytes = []byte{}
	}
	a.bytes = append(a.bytes, b...)
	id := uint64(TypeID(t))
	return Value{vTypeArena | uint64(uintptr(unsafe.Pointer(a))), id<<32 | uint64(len(a.offsets)-1)}
}

func (a *Arena) NewFromValues(t Type, values []Value) Value {
	off := uint32(len(a.values))
	if values == nil {
		off = arenaNullOffset
	}
	a.offsets = append(a.offsets, off)
	a.lengths = append(a.lengths, uint32(len(values)))
	a.values = append(a.values, values...)
	id := uint64(TypeID(t))
	return Value{vTypeArena | uint64(uintptr(unsafe.Pointer(a))), arenaUseValues | id<<32 | uint64(len(a.offsets)-1)}
}

func NewUint(t Type, x uint64) Value {
	return Value{uint64(vTypePrimitive | t.ID()), x}
}

func NewInt(t Type, x int64) Value {
	return Value{uint64(vTypePrimitive | t.ID()), uint64(x)}
}

func NewFloat(t Type, x float64) Value {
	return Value{uint64(vTypePrimitive | t.ID()), uint64(math.Float64bits(x))}
}

func NewBool(x bool) Value {
	return Value{uint64(vTypePrimitive | IDBool), boolToUint64(x)}
}

func (a *Arena) NewBytes(x []byte) Value {
	if len(x) > 15 || x == nil {
		return a.NewFromBytes(TypeBytes, x)
	}
	return newBytesOrString(vTypeBytes, x)
}

func (a *Arena) NewString(x string) Value {
	if len(x) > 15 {
		return a.NewFromBytes(TypeString, []byte(x))
	}
	return newBytesOrString(vTypeString, []byte(x))
}

func newBytesOrString(vMaskValue uint64, x []byte) Value {
	if vMaskValue != vTypeBytes && vMaskValue != vTypeString {
		panic(vMaskValue)
	}
	var b [16]byte
	copy(b[1:], x)
	a := binary.BigEndian.Uint64(b[:])
	d := binary.BigEndian.Uint64(b[8:])
	return Value{vMaskValue | uint64(len(x))<<56 | a, d}
}

func (a *Arena) Type(d uint64) Type {
	if a.zctx == nil {
		panic("a.zctx==nil with d==" + strconv.FormatUint(d, 16))
	}
	t, err := a.zctx.LookupType(arenaDescTypeID(d))
	if err != nil {
		panic(err)
	}
	return t
}

func (a *Arena) Bytes(d uint64) zcode.Bytes {
	slot := arenaDescSlot(d)
	start := a.offsets[slot]
	if start == arenaNullOffset {
		return nil
	}
	end := start + a.lengths[slot]
	if !arenaDescValues(d) {
		return a.bytes[start:end]
	}
	b := zcode.Bytes{}
	for _, val := range a.values[start:end] {
		b = zcode.Append(b, val.Bytes())
	}
	return b
}

type Value struct {
	a uint64
	d uint64
}

func (v Value) String() string {
	s := fmt.Sprintf("{0x%x,0x%x}", v.a, v.d)
	switch v.a & vTypeMask {
	case vTypeArena:
		s += " arena " + fmt.Sprintf("%#v", v.arena().Type(v.d))
		return s
	case vTypePrimitive:
		return s + " primitive " + PrimitiveName(idToType[v.a&vPrimitiveTypeIDMask])
	case vTypePrimitiveNull:
		return s + " null " + PrimitiveName(idToType[v.a&vPrimitiveTypeIDMask])
	case vTypeString:
		var b [16]byte
		binary.BigEndian.PutUint64(b[:8], v.a)
		binary.BigEndian.PutUint64(b[8:], v.d)
		length := (v.a & vLengthMask) >> 56
		return s + " string " + string(b[1:1+length])
	case vTypeBytes:
		var b [16]byte
		binary.BigEndian.PutUint64(b[:8], v.a)
		binary.BigEndian.PutUint64(b[8:], v.d)
		length := (v.a & vLengthMask) >> 56
		return s + " bytes " + fmt.Sprintf("%v", b[1:1+length])
	}
	panic(v)
}

const (
	vTypeNull            = 0 << 60
	vTypeArena           = 1 << 60
	vTypePrimitive       = 2 << 60
	vTypePrimitiveNull   = 3 << 60
	vTypeBytes           = 4 << 60
	vTypeString          = 5 << 60
	vTypeMask            = uint64(0xf) << 60
	vLengthMask          = uint64(0x0f) << 56
	vPrimitiveTypeIDMask = 0xff
)

func (v Value) CheckArena(a *Arena) {
	if a2, ok := v.Arena(); ok && a2 != a {
		panic("arena mismatch")
	}
}

func (v Value) Arena() (*Arena, bool) {
	if v.a&vTypeMask != vTypeArena {
		return nil, false
	}
	return (*Arena)(unsafe.Pointer(uintptr(v.a & ^vTypeMask))), true
}

func (v Value) arena() *Arena {
	if v.a&vTypeMask != vTypeArena {
		panic(v)
	}
	return (*Arena)(unsafe.Pointer(uintptr(v.a & ^vTypeMask)))
}

func (v Value) CopyToArena(a *Arena) Value {
	if v.a&vTypeMask != vTypeArena {
		return v
	}
	return a.NewValue(v.Type(), v.Bytes())
}

func (v Value) Type() Type {
	switch v.a & vTypeMask {
	case vTypeArena:
		return v.arena().Type(v.d)
	case vTypePrimitive, vTypePrimitiveNull:
		return idToType[v.a&vPrimitiveTypeIDMask]
	case vTypeBytes:
		return TypeBytes
	case vTypeString:
		return TypeString
	}
	panic(v)
}

func (v Value) typeID() int {
	switch v.a & vTypeMask {
	case vTypeArena:
		return v.arena().Type(v.d).ID()
	case vTypePrimitive, vTypePrimitiveNull:
		return int(v.a & vPrimitiveTypeIDMask)
	case vTypeBytes:
		return IDBytes
	case vTypeString:
		return IDString
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

// Uint returns v's underlying value.  It panics if v's underlying type is not
// TypeUint8, TypeUint16, TypeUint32, or TypeUint64.
func (v Value) Uint() uint64 {
	if !IsUnsigned(v.typeID()) {
		panic(fmt.Sprintf("zed.Value.Uint called on %T", v.Type()))
	}
	if v.a&vTypeMask == vTypePrimitive {
		return v.d
	}
	return DecodeUint(v.arena().Bytes(v.d))
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
	return DecodeInt(v.arena().Bytes(v.d))
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
	return DecodeFloat(v.arena().Bytes(v.d))
}

// Bool returns v's underlying value.  It panics if v's underlying type is not
// TypeBool.
func (v Value) Bool() bool {
	if v.typeID() != IDBool {
		panic(fmt.Sprintf("zed.Value.Bool called on %T", v.Type))
	}
	if v.a&vTypeMask == vTypePrimitive {
		return v.d != 0
	}
	return DecodeBool(v.arena().Bytes(v.d))
}

func (v Value) Bytes() zcode.Bytes {
	switch v.a & vTypeMask {
	case vTypeArena:
		return v.arena().Bytes(v.d)
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

func (v Value) IsNull() bool {
	return v.a&vTypeMask == vTypeNull ||
		v.a&vTypeMask == vTypePrimitiveNull ||
		v.a&vTypeMask == vTypeArena && v.arena().offsets[arenaDescSlot(v.d)] == arenaNullOffset
}
