package zed

import (
	"encoding/binary"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/brimdata/zed/zcode"
)

type Arena struct {
	pool *sync.Pool
	refs int32

	byID              []Type
	offsetsAndLengths []uint64
	bytes             []byte
	values            []Value
	free              func()
}

var (
	arenaPool          sync.Pool
	arenaWithBytesPool sync.Pool
)

func NewArena() *Arena {
	return newArena(&arenaPool)
}

func NewArenaWithBytes(bytes []byte, free func()) *Arena {
	a := newArena(&arenaWithBytesPool)
	a.bytes = bytes
	a.free = free
	return a
}

func newArena(pool *sync.Pool) *Arena {
	a, ok := pool.Get().(*Arena)
	if ok {
		clear(a.byID)
		a.Reset()
	} else {
		a = &Arena{pool: pool}
	}
	a.refs = 1
	if a.bytes == nil {
		a.bytes = []byte{}
	}
	return a
}

func (a *Arena) Ref() { atomic.AddInt32(&a.refs, 1) }

func (a *Arena) Unref() {
	if refs := atomic.AddInt32(&a.refs, -1); refs == 0 {
		if a.free != nil {
			a.bytes = nil
			a.free()
		}
		a.pool.Put(a)
	} else if refs < 0 {
		panic("negative arena reference count")
	}
}

func (a *Arena) Reset() {
	a.offsetsAndLengths = a.offsetsAndLengths[:0]
	a.bytes = a.bytes[:0]
	a.values = a.values[:0]
}

func (a *Arena) New(t Type, b zcode.Bytes) Value {
	offset := len(a.bytes)
	if b == nil {
		offset = arenaNullOffset
	}
	a.bytes = append(a.bytes, b...)
	return a.new(t, offset, len(b), 0)
}

func (a *Arena) NewFromOffsetAndLength(t Type, offset, length int) Value {
	return a.new(t, offset, length, 0)
}

func (a *Arena) NewFromValues(t Type, values []Value) Value {
	offset := len(a.values)
	if values == nil {
		offset = arenaNullOffset
	}
	a.values = append(a.values, values...)
	return a.new(t, offset, len(values), arenaUseValues)
}

func (a *Arena) new(t Type, offset, length int, dFlags uint64) Value {
	if uint64(offset) > math.MaxUint32 {
		panic("bad offset " + strconv.Itoa(offset))
	}
	if uint64(length) > math.MaxUint32 {
		panic("bad length " + strconv.Itoa(length))
	}
	id := TypeID(t)
	if id >= len(a.byID) {
		a.byID = slices.Grow(a.byID[:0], id+1)[:id+1]
	}
	if tt := a.byID[id]; tt == nil {
		a.byID[id] = t
	} else if tt != t {
		panic("type conflict")
	}
	a.offsetsAndLengths = append(a.offsetsAndLengths, uint64(offset)<<32|uint64(length))
	return Value{vTypeArena | uint64(uintptr(unsafe.Pointer(a))), dFlags | uint64(id)<<32 | uint64(len(a.offsetsAndLengths)-1)}
}

func (a *Arena) NewBytes(x []byte) Value {
	if len(x) < 16 {
		if x == nil {
			return NullBytes
		}
		return newNativeBytes(vTypeBytes, x)
	}
	return a.New(TypeBytes, x)
}

func (a *Arena) NewString(x string) Value {
	if len(x) < 16 {
		return newNativeBytes(vTypeString, []byte(x))
	}
	return a.New(TypeString, []byte(x))
}

func newNativeBytes(vMaskValue uint64, x []byte) Value {
	var b [16]byte
	copy(b[1:], x)
	a := binary.BigEndian.Uint64(b[:])
	d := binary.BigEndian.Uint64(b[8:])
	return Value{vMaskValue | uint64(len(x))<<56 | a, d}
}

func (a *Arena) NewIP(x netip.Addr) Value {
	return a.New(TypeIP, EncodeIP(x))
}

func (a *Arena) NewNet(p netip.Prefix) Value {
	return a.New(TypeNet, EncodeNet(p))
}

func (a *Arena) NewTypeValue(t Type) Value {
	return a.New(TypeNet, EncodeTypeValue(t))
}

func (a *Arena) type_(d uint64) Type {
	return a.byID[arenaDescTypeID(d)]
}

func (a *Arena) bytes_(d uint64) zcode.Bytes {
	slot := arenaDescSlot(d)
	start := a.offsetsAndLengths[slot] >> 32
	if start == arenaNullOffset {
		return nil
	}
	end := start + a.offsetsAndLengths[slot]&0xffffffff
	if !arenaDescValues(d) {
		return a.bytes[start:end]
	}
	if union, ok := TypeUnder(a.type_(d)).(*TypeUnion); ok {
		val := a.values[start]
		tag := union.TagOf(val.Type())
		b := zcode.Append(nil, EncodeInt(int64(tag)))
		return zcode.Append(b, val.Bytes())
	}
	b := zcode.Bytes{}
	for _, val := range a.values[start:end] {
		b = zcode.Append(b, val.Bytes())
	}
	return b
}
