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
		//XXX this doesn't look like it could work right because we
		// need to know whether the sub-zval is a container or a value,
		// but in practice, the lengths are always the same because the
		// variable length encoding of the size is not affected by the
		// low bit of the tag encoding.
		n += sizeOfValue(len(v))
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

// AppendUvarint is like encoding/binary.PutUvarint but appends to dst instead
// of writing into it.
func AppendUvarint(dst []byte, u64 uint64) []byte {
	for u64 >= 0x80 {
		dst = append(dst, byte(u64)|0x80)
		u64 >>= 7
	}
	return append(dst, byte(u64))
}

// sizeOfUvarint returns the number of bytes required by AppendUvarint to
// represent u64.
func sizeOfUvarint(u64 uint64) int {
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

// sizeOfContainer returns the number of bytes required to represent
// a container byte slice of the indicated length as a zval.
func sizeOfContainer(length int) int {
	return (sizeOfUvarint(containerTag(length))) + length
}

// sizeOfValue returns the number of bytes required to represent
// a byte slice of the indicated length as a zval.
func sizeOfValue(length int) int {
	return int(sizeOfUvarint(valueTag(length))) + length
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
