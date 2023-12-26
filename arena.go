package zed

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"

	"github.com/brimdata/zed/zcode"
)

type Arena struct {
	zctx *Context

	offsets []uint32
	lengths []uint32
	bytes   []byte
	values  []value
}

const arenaNullOffset = math.MaxUint32
const arenaUseValues = uint64(1) << 63

func arenaDescSlot(d uint64) int    { return int(d & math.MaxUint32) }
func arenaDescTypeID(d uint64) int  { return int(d >> 32 &^ arenaUseValues) }
func arenaDescValues(d uint64) bool { return d&arenaUseValues != 0 }

func (a *Arena) NewFromBytes(t Type, b zcode.Bytes) value {
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
	return value{uint64(uintptr(unsafe.Pointer(a))), id<<32 | uint64(len(a.offsets)-1)}
}

func (a *Arena) NewFromValues(t Type, values []value) value {
	off := uint32(len(a.values))
	if values == nil {
		off = arenaNullOffset
	}
	a.offsets = append(a.offsets, off)
	a.lengths = append(a.lengths, uint32(len(values)))
	a.values = append(a.values, values...)
	id := uint64(TypeID(t))
	return value{uint64(uintptr(unsafe.Pointer(a))), arenaUseValues | id<<32 | uint64(len(a.offsets)-1)}
}

func (a *Arena) NewUint(t Type, x uint64) value {
	return value{uint64(vPrimitive | t.ID()), x}
}

func (a *Arena) NewInt(t Type, x int64) value {
	return value{uint64(vPrimitive | t.ID()), uint64(x)}
}

func (a *Arena) NewFloat(t Type, x float64) value {
	return value{uint64(vPrimitive | t.ID()), uint64(math.Float64bits(x))}
}

func (a *Arena) NewBool(x bool) value {
	return value{uint64(vPrimitive | IDBool), boolToUint64(x)}
}

func (a *Arena) NewBytes(x []byte) value {
	if len(x) > 15 || x == nil {
		return a.NewFromBytes(TypeBytes, x)
	}
	v := newBytes(x)
	v.a |= vBytes
	return v
}

func (a *Arena) NewString(x string) value {
	if len(x) > 15 {
		return a.NewFromBytes(TypeString, []byte(x))
	}
	v := newBytes([]byte(x))
	v.a |= vString
	return v
}

func newBytes(x []byte) value {
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
	return value{a, d}
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
	if arenaDescValues(d) {
		return a.bytes[start:end]
	}
	b := zcode.Bytes{}
	for _, val := range a.values[start:end] {
		b = zcode.Append(b, val.Bytes())
	}
	return b
}

type value struct {
	a uint64
	d uint64
}

const (
	vArena               = 0 << 60
	vPrimitive           = 1 << 60
	vBytes               = 2 << 60
	vString              = 3 << 60
	vMask                = uint64(0xf) << 60
	vPrimitiveTypeIDMask = 0xff
)

func (v value) arena() *Arena { return (*Arena)(unsafe.Pointer(uintptr(v.a))) }

func (v value) Type() Type {
	switch v.a & vMask {
	case vArena:
		return v.arena().Type(v.d)
	case vPrimitive:
		return idToType[v.a&vPrimitiveTypeIDMask]
	case vBytes:
		return TypeBytes
	case vString:
		return TypeString
	}
	panic(v)
}

func (v value) typeID() int {
	switch v.a & vMask {
	case vArena:
		return v.arena().Type(v.d).ID()
	case vPrimitive:
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
func (v value) Uint() uint64 {
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
func (v value) Int() int64 {
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
func (v value) Float() float64 {
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
func (v value) Bool() bool {
	if v.typeID() != IDBool {
		panic(fmt.Sprintf("zed.Value.Bool called on %T", v.Type))
	}
	if v.a&vMask == vPrimitive {
		return v.d != 0
	}
	return DecodeBool(v.arena().Bytes(v.d))
}

func (v value) Bytes() zcode.Bytes {
	switch v.a & vMask {
	case vArena:
		return v.arena().Bytes(v.d)
	case vString, vBytes:
		var b [16]byte
		binary.LittleEndian.PutUint64(b[:8], v.a)
		binary.LittleEndian.PutUint64(b[8:], v.d)
		length := (v.a & ^vMask) >> 60
		return b[1:length]
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
	}
	panic(v)
}

func (v value) IsNull() bool {
	return v.a&vMask == vArena && v.arena().offsets[arenaDescSlot(v.d)] == arenaNullOffset
}
