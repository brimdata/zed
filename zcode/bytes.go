// Package zcode implements serialization and deserialzation for ZNG values.
//
// Values of primitive type are represented by an unsigned integer tag and an
// optional byte-sequence body.  A tag of zero indicates that the value is
// unset, and no body follows.  A nonzero tag indicates that the value is set,
// and the value itself follows as a body of length tag-1.
//
// Values of container type (record, set, or vector) are represented similarly,
// with the body containing a sequence of zero or more serialized values.
package zcode

import (
	"encoding/binary"
	"errors"
)

var (
	ErrNotContainer = errors.New("not a container")
	ErrNotSingleton = errors.New("not a single container")
)

// Bytes is the serialized representation of a sequence of ZNG values.
type Bytes []byte

// Iter returns an Iter for the receiver.
func (b Bytes) Iter() Iter {
	return Iter(b)
}

// String returns a string representation of the receiver.
func (b Bytes) String() string {
	buf, err := b.build(nil)
	if err != nil {
		panic("zcode encoding has bad format: " + err.Error())
	}
	return string(buf)
}

const hex = "0123456789abcdef"

func appendBytes(b, v []byte) []byte {
	first := true
	for _, c := range v {
		if !first {
			b = append(b, ' ')
		} else {
			first = false
		}
		b = append(b, hex[c>>4])
		b = append(b, hex[c&0xf])
	}
	return b
}

func (b Bytes) build(dst []byte) ([]byte, error) {
	for it := b.Iter(); !it.Done(); {
		v, container, err := it.Next()
		if err != nil {
			return nil, err
		}
		if container {
			if v == nil {
				dst = append(dst, "(*)"...)
				continue
			}
			dst = append(dst, '[')
			dst, err = v.build(dst)
			if err != nil {
				return nil, err
			}
			dst = append(dst, ']')
		} else {
			dst = append(dst, '(')
			dst = appendBytes(dst, v)
			dst = append(dst, ')')
		}
	}
	return dst, nil
}

// ContainerBody returns the body of the receiver, which must hold a single
// container.  If the receiver is not a container, ErrNotContainer is returned.
// If the receiver is not a single container, ErrNotSingleton is returned.
func (b Bytes) ContainerBody() (Bytes, error) {
	it := b.Iter()
	body, container, err := it.Next()
	if err != nil {
		return nil, err
	}
	if !container {
		return nil, ErrNotContainer
	}
	if !it.Done() {
		return nil, ErrNotSingleton
	}
	return body, nil
}

// AppendContainer appends val to dst as a container value and returns the
// extended buffer.
func AppendContainer(dst Bytes, val Bytes) Bytes {
	if val == nil {
		return AppendUvarint(dst, containerTagUnset)
	}
	dst = AppendUvarint(dst, containerTag(len(val)))
	dst = append(dst, val...)
	return dst
}

// AppendPrimitive appends val to dst as a primitive value and returns the
// extended buffer.
func AppendPrimitive(dst Bytes, val []byte) Bytes {
	if val == nil {
		return AppendUvarint(dst, primitiveTagUnset)
	}
	dst = AppendUvarint(dst, primitiveTag(len(val)))
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

// sizeOfUvarint returns the number of bytes required by appendUvarint to
// represent u64.
func sizeOfUvarint(u64 uint64) int {
	n := 1
	for u64 >= 0x80 {
		n++
		u64 >>= 7
	}
	return n
}

// uvarint just calls binary.Uvarint.  It's here for symmetry with
// appendUvarint.
func uvarint(buf []byte) (uint64, int) {
	return binary.Uvarint(buf)
}

func containerTag(length int) uint64 {
	return (uint64(length)+1)<<1 | 1
}

func primitiveTag(length int) uint64 {
	return (uint64(length) + 1) << 1
}

const (
	primitiveTagUnset = 0
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
