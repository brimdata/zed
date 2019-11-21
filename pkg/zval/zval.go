// Package zval implements serialization and deserialzation for zson values.
//
// Values of primitive type are represented by an unsigned integer tag and an
// optional byte sequence.  A tag of zero indicates that the value is unset, and
// no byte sequence follows.  A nonzero tag indicates that the value is set, and
// the value itself follows as a byte sequence of length tag-1.
//
// Values of container type (record, set, or vector) are represented similarly,
// with the byte sequence containing a sequence of zero or more serialized
// values.
package zval

import (
	"encoding/binary"
	"fmt"
)

const (
	beginContainer = -1
	endContainer   = -2
)

type node struct {
	innerLen int
	outerLen int
	dfs      int
}

// Builder implements an API for holding an intermediate representation
// of a hierarchical set of values arranged in a tree, e.g., structured
// values that can contain nested and recursive aggregate values.
// We encode a DFS traversal in a flat data structure that can be
// reused across invocations so we don't otherwise allocate a tree
// data structure for every record parsed that would then be GC'd.
type Builder struct {
	nodes  []node
	leaves [][]byte
}

func NewBuilder() *Builder {
	return &Builder{
		nodes:  make([]node, 0, 64),
		leaves: make([][]byte, 0, 64),
	}
}

func (b *Builder) Reset() {
	b.nodes = b.nodes[:0]
	b.leaves = b.leaves[:0]
}

func (b *Builder) Begin() {
	b.nodes = append(b.nodes, node{dfs: beginContainer})
}

func (b *Builder) End() {
	b.nodes = append(b.nodes, node{dfs: endContainer})
}

func (b *Builder) Append(leaf []byte) {
	k := len(b.leaves)
	b.leaves = append(b.leaves, leaf)
	b.nodes = append(b.nodes, node{dfs: k})
}

func (b *Builder) measure(off int) int {
	node := &b.nodes[off]
	dfs := node.dfs
	if dfs == beginContainer {
		// skip over start token
		off++
		for off < len(b.nodes) {
			next := b.measure(off)
			if next < 0 {
				// skip over end token
				off++
				break
			}
			node.innerLen += b.nodes[off].outerLen
			off = next
		}
		node.outerLen = sizeOfContainer(node.innerLen)
		return off
	}
	if dfs == endContainer {
		return -1
	}
	n := len(b.leaves[dfs])
	node.innerLen = n
	node.outerLen = sizeOfValue(n)
	return off + 1
}

func (b *Builder) encode(dst []byte, off int) ([]byte, int) {
	node := &b.nodes[off]
	dfs := node.dfs
	if dfs == beginContainer {
		// skip over start token
		off++
		if b.nodes[off].dfs == endContainer {
			return AppendUvarint(dst, containerTagUnset), off + 1
		}
		dst = AppendUvarint(dst, containerTag(node.innerLen))
		for off < len(b.nodes) {
			var next int
			dst, next = b.encode(dst, off)
			if next < 0 {
				// skip over end token
				off++
				break
			}
			off = next
		}
		return dst, off
	}
	if dfs == endContainer {
		return dst, -1
	}
	return AppendValue(dst, b.leaves[dfs]), off + 1
}

func (b *Builder) Encode() []byte {
	off := 0
	for off < len(b.nodes) {
		next := b.measure(off)
		if next < 0 {
			break
		}
		off = next
	}
	off = 0
	var zv []byte
	for off < len(b.nodes) {
		var next int
		zv, next = b.encode(zv, off)
		if next < 0 {
			break
		}
		off = next
	}
	return zv
}

// Iter iterates over a sequence of zvals.
type Iter []byte

// Done returns true if no zvals remain.
func (i *Iter) Done() bool {
	return len(*i) == 0
}

// Next returns the next zval.  It returns an empty slice for an empty or
// zero-length zval and nil for an unset zval.
func (i *Iter) Next() ([]byte, bool, error) {
	// Uvarint is zero for an unset zval; otherwise, it is the value's
	// length plus one.
	u64, n := Uvarint(*i)
	if n <= 0 {
		return nil, false, fmt.Errorf("bad uvarint: %d", n)
	}
	if tagIsUnset(u64) {
		*i = (*i)[n:]
		return nil, tagIsContainer(u64), nil
	}
	end := n + tagLength(u64)
	val := (*i)[n:end]
	*i = (*i)[end:]
	return val, tagIsContainer(u64), nil
}

// AppendContainer appends to dst a zval container comprising the zvals in vals.
func AppendContainer(dst []byte, vals [][]byte) []byte {
	if vals == nil {
		return AppendUvarint(dst, containerTagUnset)
	}
	var n int
	for _, v := range vals {
		n += sizeBytes(v)
	}
	dst = AppendUvarint(dst, containerTag(n))
	for _, v := range vals {
		dst = AppendValue(dst, v)
	}
	return dst
}

// AppendValue appends to dst the zval in val.
func AppendValue(dst []byte, val []byte) []byte {
	if val == nil {
		return AppendUvarint(dst, valueTagUnset)
	}
	dst = AppendUvarint(dst, valueTag(len(val)))
	return append(dst, val...)
}

// sizeBytes returns the number of bytes required by AppendValue to represent
// the zval in val.
func sizeBytes(val []byte) int {
	// This really is correct even when val is nil.
	return sizeUvarint(valueTag(len(val))) + len(val)
}

// AppendUvarint is like encoding/binary.PutUvarint but appends to dst instead
// of writing into it.
func AppendUvarint(dst []byte, u64 uint64) []byte {
	for u64 >= 0x80 {
		dst = append(dst, byte(u64)|0x80)
		u64 >>= 7
	}
	return append(dst, byte(u64))
}

// sizeUvarint returns the number of bytes required by AppendUvarint to
// represent u64.
func sizeUvarint(u64 uint64) int {
	n := 1
	for u64 >= 0x80 {
		n++
		u64 >>= 7
	}
	return n
}

// Uvarint just calls binary.Uvarint.  It's here for symmetry with
// AppendUvarint.
func Uvarint(buf []byte) (uint64, int) {
	return binary.Uvarint(buf)
}

func sizeOfContainer(length int) int {
	return (sizeUvarint(containerTag(length))) + length
}

func sizeOfValue(length int) int {
	return int(sizeUvarint(valueTag(length))) + length
}

func containerTag(length int) uint64 {
	return (uint64(length)+1)<<1 | 1
}

func valueTag(length int) uint64 {
	return (uint64(length) + 1) << 1
}

const (
	valueTagUnset     = 0
	containerTagUnset = 1
)

func tagIsContainer(t uint64) bool {
	return t&1 == 1
}

func tagIsUnset(t uint64) bool {
	return t>>1 == 0
}

func tagLength(t uint64) int {
	return int(t>>1 - 1)
}
