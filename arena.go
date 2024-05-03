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

// Arena is an reference-counted allocator for Values. Two constraints govern
// its use.  First, the Type of each Value in an arena must belong to a single
// Context.  Second, an arena must be reachable at any point in a program where
// its Values are accessed.
type Arena struct {
	pool *sync.Pool
	refs int32

	byID              []Type
	offsetsAndLengths []uint64

	buf    []byte
	free   func()
	slices [][]byte
	values []Value
}

var (
	arenaPool           sync.Pool
	arenaWithBufferPool sync.Pool
)

// NewArena returns an empty arena.
func NewArena() *Arena {
	return newArena(&arenaPool)
}

// NewArenaWithBuffer returns an arena whose buffer is initialized to buf.  If
// free is not nil, it is called when Unref decrements the arena's reference
// count to zero and can be used to return buf to an allocator.
func NewArenaWithBuffer(buf []byte, free func()) *Arena {
	a := newArena(&arenaWithBufferPool)
	a.buf = buf
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
	if a.buf == nil {
		a.buf = []byte{}
	}
	return a
}

func (a *Arena) Ref() {
	atomic.AddInt32(&a.refs, 1)
}

func (a *Arena) Unref() {
	if refs := atomic.AddInt32(&a.refs, -1); refs == 0 {
		if a.free != nil {
			a.buf = nil
			a.free()
		}
		a.pool.Put(a)
	} else if refs < 0 {
		panic("negative arena reference count")
	}
}

func (a *Arena) Reset() {
	a.offsetsAndLengths = a.offsetsAndLengths[:0]
	a.buf = a.buf[:0]
	a.slices = a.slices[:0]
	a.values = a.values[:0]
}

// New returns a new Value in a whose bytes are given by b.  a retains a
// reference to b, so modifying b also modifies the value.
func (a *Arena) New(t Type, b zcode.Bytes) Value {
	if b == nil {
		return a.new(t, dStorageNull, 0, 0)
	}
	offset := len(a.slices)
	a.slices = append(a.slices, b)
	return a.new(t, dStorageSlices, offset, 0)
}

// NewFromOffsetAndLength returns a new Value in a whose bytes are a slice of
// a's buffer from offset to offet+length.  NewFromOffsetAndLength is intended
// for use with arenas obtained from NewArenaWithBuffer.
func (a *Arena) NewFromOffsetAndLength(t Type, offset, length int) Value {
	return a.new(t, dStorageBuffer, offset, length)
}

// NewFromValues returns a new Value in a that is a record, array, set, map,
// union, or error with values as its constituent values.  NewFromValues does
// not retain a reference to values.
func (a *Arena) NewFromValues(t Type, values []Value) Value {
	if values == nil {
		return a.new(t, dStorageNull, 0, 0)
	}
	offset := len(a.values)
	a.values = append(a.values, values...)
	return a.new(t, dStorageValues, offset, len(values))
}

func (a *Arena) new(t Type, dStorage uint64, offset, length int) Value {
	id := TypeID(t)
	if id >= len(a.byID) {
		a.byID = slices.Grow(a.byID[:0], id+1)[:id+1]
	}
	if tt := a.byID[id]; tt == nil {
		a.byID[id] = t
	} else if tt != t {
		panic("type conflict")
	}
	var slot int
	if dStorage != dStorageNull {
		if uint64(offset) > math.MaxUint32 {
			panic("bad offset " + strconv.Itoa(offset))
		}
		if uint64(length) > math.MaxUint32 {
			panic("bad length " + strconv.Itoa(length))
		}
		a.offsetsAndLengths = append(a.offsetsAndLengths, uint64(offset)<<32|uint64(length))
		slot = len(a.offsetsAndLengths) - 1
	}
	return Value{aReprArena | uint64(uintptr(unsafe.Pointer(a))), dStorage | uint64(id)<<32 | uint64(slot)}
}

// NewBytes returns a new Value in a of type TypeBytes.  a may retain a
// reference to b, so modifying b may also modify the value.
func (a *Arena) NewBytes(b []byte) Value {
	if len(b) < 16 {
		if b == nil {
			return NullBytes
		}
		return newSmallBytesOrString(aReprSmallBytes, b)
	}
	return a.New(TypeBytes, b)
}

// NewString returns a new Value in a of type TypeString.
func (a *Arena) NewString(s string) Value {
	if len(s) < 16 {
		return newSmallBytesOrString(aReprsSmallString, []byte(s))
	}
	return a.New(TypeString, []byte(s))
}

func newSmallBytesOrString(aRepr uint64, b []byte) Value {
	var bb [16]byte
	copy(bb[1:], b)
	a := binary.BigEndian.Uint64(bb[:])
	d := binary.BigEndian.Uint64(bb[8:])
	return Value{aRepr | uint64(len(b))<<56 | a, d}
}

// NewIP returns a new Value in a of type TypeIP.
func (a *Arena) NewIP(A netip.Addr) Value {
	return a.New(TypeIP, EncodeIP(A))
}

// NewNet returns a new Value in a of type TypeNet.
func (a *Arena) NewNet(p netip.Prefix) Value {
	return a.New(TypeNet, EncodeNet(p))
}

func (a *Arena) type_(d uint64) Type {
	return a.byID[d&^dStorageMask>>32]
}

func (a *Arena) bytes_(d uint64) zcode.Bytes {
	switch d & dStorageMask {
	case dStorageBuffer:
		offset, length := a.offsetAndLength(d)
		return a.buf[offset : offset+length]
	case dStorageNull:
		return nil
	case dStorageSlices:
		offset, _ := a.offsetAndLength(d)
		return a.slices[offset]
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
