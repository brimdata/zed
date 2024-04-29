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
	if b == nil {
		return a.new(t, 0, 0, dStorageNull)
	}
	offset := len(a.bytes)
	a.bytes = append(a.bytes, b...)
	return a.new(t, offset, len(b), dStorageBytes)
}

func (a *Arena) NewFromOffsetAndLength(t Type, offset, length int) Value {
	return a.new(t, offset, length, dStorageBytes)
}

func (a *Arena) NewFromValues(t Type, values []Value) Value {
	if values == nil {
		return a.new(t, 0, 0, dStorageNull)
	}
	offset := len(a.values)
	a.values = append(a.values, values...)
	return a.new(t, offset, len(values), dStorageValues)
}

func (a *Arena) new(t Type, offset, length int, dStorage uint64) Value {
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
	return Value{aTypeArena | uint64(uintptr(unsafe.Pointer(a))), dStorage | uint64(id)<<32 | uint64(len(a.offsetsAndLengths)-1)}
}

func (a *Arena) NewBytes(x []byte) Value {
	if len(x) < 16 {
		if x == nil {
			return NullBytes
		}
		return newNativeBytes(aTypeBytes, x)
	}
	return a.New(TypeBytes, x)
}

func (a *Arena) NewString(x string) Value {
	if len(x) < 16 {
		return newNativeBytes(aTypeString, []byte(x))
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
	return a.byID[d&^dStorageMask>>32]
}

func (a *Arena) bytes_(d uint64) zcode.Bytes {
	switch d & dStorageMask {
	case dStorageBytes:
		offset, length := a.offsetAndLength(d)
		return a.bytes[offset : offset+length]
	case dStorageNull:
		return nil
	case dStorageValues:
		offset, length := a.offsetAndLength(d)
		if union, ok := TypeUnder(a.type_(d)).(*TypeUnion); ok {
			val := a.values[offset]
			tag := union.TagOf(val.Type())
			b := zcode.Append(nil, EncodeInt(int64(tag)))
			return zcode.Append(b, val.Bytes())
		}
		b := zcode.Bytes{}
		for _, val := range a.values[offset : offset+length] {
			b = zcode.Append(b, val.Bytes())
		}
		return b
	}
	panic(d)
}

func (a *Arena) offsetAndLength(d uint64) (uint64, uint64) {
	slot := d & math.MaxUint32
	offset := a.offsetsAndLengths[slot] >> 32
	length := a.offsetsAndLengths[slot] & 0xffffffff
	return offset, length
}
