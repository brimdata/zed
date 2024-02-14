package zed

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
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
	return Value{vArena | uint64(uintptr(unsafe.Pointer(a))), id<<32 | uint64(len(a.offsets)-1)}
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
	return Value{vArena | uint64(uintptr(unsafe.Pointer(a))), arenaUseValues | id<<32 | uint64(len(a.offsets)-1)}
}

func NewUint(t Type, x uint64) Value {
	return Value{uint64(vPrimitive | t.ID()), x}
}

func NewInt(t Type, x int64) Value {
	return Value{uint64(vPrimitive | t.ID()), uint64(x)}
}

func NewFloat(t Type, x float64) Value {
	return Value{uint64(vPrimitive | t.ID()), uint64(math.Float64bits(x))}
}

func NewBool(x bool) Value {
	return Value{uint64(vPrimitive | IDBool), boolToUint64(x)}
}

func (a *Arena) NewBytes(x []byte) Value {
	if len(x) > 15 || x == nil {
		return a.NewFromBytes(TypeBytes, x)
	}
	v := newBytes(x)
	v.a |= vBytes
	return v
}

func (a *Arena) NewString(x string) Value {
	if len(x) > 15 {
		return a.NewFromBytes(TypeString, []byte(x))
	}
	v := newBytes([]byte(x))
	v.a |= vString
	return v
}

func newBytes(x []byte) Value {
	a := uint64(len(x))
	for i := 0; i < 7; i++ {
		a <<= 8
		if i < len(x) {
			a |= uint64(x[i])
		}
	}
	var d uint64
	if len(x) > 7 {
		for i := 7; i < 14; i++ {
			d <<= 8
			if i < len(x) {
				d |= uint64(x[i])
			}
		}
	}
	return Value{a, d}
}

func (a *Arena) Type(d uint64) Type {
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
	s := fmt.Sprintf("%0x %0x", v.a, v.d)
	switch v.a & vMask {
	case vArena:
		s += " arena " + fmt.Sprintf("%#v", v.arena().Type(v.d))
		return s
	case vString:
		var b [16]byte
		binary.LittleEndian.PutUint64(b[:8], v.a)
		binary.LittleEndian.PutUint64(b[8:], v.d)
		length := (v.a & ^vMask) >> 60
		return s + " string " + string(b[1:1+length])
	case vBytes:
		var b [16]byte
		binary.LittleEndian.PutUint64(b[:8], v.a)
		binary.LittleEndian.PutUint64(b[8:], v.d)
		length := (v.a & ^vMask) >> 60
		return s + " bytes " + fmt.Sprintf("%v", b[1:1+length])
	case vPrimitive:
		return s + " primitive " + PrimitiveName(idToType[v.a&vPrimitiveTypeIDMask])
	case vPrimitiveNull:
		return s + " null " + PrimitiveName(idToType[v.a&vPrimitiveTypeIDMask])
	}
	panic(v)
}

const (
	vNull                = 0 << 60
	vArena               = 1 << 60
	vPrimitive           = 2 << 60
	vPrimitiveNull       = 3 << 60
	vBytes               = 4 << 60
	vString              = 5 << 60
	vMask                = uint64(0xf) << 60
	vPrimitiveTypeIDMask = 0xff
)

func (v Value) CheckArena(a *Arena) {
	if a2, ok := v.Arena(); ok && a2 != a {
		panic("arena mismatch")
	}
}

func (v Value) Arena() (*Arena, bool) {
	if v.a&vMask != vArena {
		return nil, false
	}
	return (*Arena)(unsafe.Pointer(uintptr(v.a))), true
}

func (v Value) arena() *Arena {
	if v.a&vMask != vArena {
		panic(v)
	}
	return (*Arena)(unsafe.Pointer(uintptr(v.a)))
}

func (v Value) CopyToArena(a *Arena) Value {
	if v.a&vMask != vArena {
		return v
	}
	return a.NewValue(v.Type(), v.Bytes())
}

func (v Value) Type() Type {
	switch v.a & vMask {
	case vArena:
		return v.arena().Type(v.d)
	case vPrimitive, vPrimitiveNull:
		return idToType[v.a&vPrimitiveTypeIDMask]
	case vBytes:
		return TypeBytes
	case vString:
		return TypeString
	}
	panic(v)
}

func (v Value) typeID() int {
	switch v.a & vMask {
	case vArena:
		return v.arena().Type(v.d).ID()
	case vPrimitive, vPrimitiveNull:
		return int(v.a & vPrimitiveTypeIDMask)
	case vBytes:
		return IDBytes
	case vString:
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
	if v.a&vMask == vPrimitive {
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
	if v.a&vMask == vPrimitive {
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
	if v.a&vMask == vPrimitive {
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
	if v.a&vMask == vPrimitive {
		return v.d != 0
	}
	return DecodeBool(v.arena().Bytes(v.d))
}

func (v Value) Bytes() zcode.Bytes {
	switch v.a & vMask {
	case vArena:
		return v.arena().Bytes(v.d)
	case vBytes, vString:
		var b [16]byte
		binary.LittleEndian.PutUint64(b[:8], v.a)
		binary.LittleEndian.PutUint64(b[8:], v.d)
		length := (v.a & ^vMask) >> 60
		return b[1 : 1+length]
	case vPrimitive:
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
	case vPrimitiveNull:
		return nil
	}
	panic(v)
}

func (v Value) IsNull() bool {
	return v.a&vMask == vNull ||
		v.a&vMask == vPrimitiveNull ||
		v.a&vMask == vArena && v.arena().offsets[arenaDescSlot(v.d)] == arenaNullOffset
}
