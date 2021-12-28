// Package zcode implements serialization and deserialzation for ZNG values.
//
// Values of primitive type are represented by an unsigned integer tag and an
// optional byte-sequence body.  A tag of zero indicates that the value is
// null, and no body follows.  A nonzero tag indicates that the value is set,
// and the value itself follows as a body of length tag-1.
//
// Values of union type are represented similarly, with the body
// prefixed by an integer specifying the index determining the type of
// the value in reference to the union type.
//
// Values of container type (record, set, or array) are represented similarly,
// with the body containing a sequence of zero or more serialized values.
package zcode

import (
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
		if v, container := it.Next(); container {
			if v == nil {
				dst = append(dst, "(*)"...)
				continue
			}
			dst = append(dst, '[')
			var err error
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
	body, container := it.Next()
	if !container {
		return nil, ErrNotContainer
	}
	if !it.Done() {
		return nil, ErrNotSingleton
	}
	return body, nil
}

func AppendAs(dst Bytes, container bool, val []byte) Bytes {
	if container {
		return AppendContainer(dst, val)
	}
	return AppendPrimitive(dst, val)
}

// AppendContainer appends val to dst as a container value and returns the
// extended buffer.
func AppendContainer(dst Bytes, val Bytes) Bytes {
	if val == nil {
		return AppendUvarint(dst, tagNull)
	}
	dst = AppendUvarint(dst, containerTag(len(val)))
	dst = append(dst, val...)
	return dst
}

// AppendPrimitive appends val to dst as a primitive value and returns the
// extended buffer.
func AppendPrimitive(dst Bytes, val []byte) Bytes {
	if val == nil {
		return AppendUvarint(dst, tagNull)
	}
	dst = AppendUvarint(dst, primitiveTag(len(val)))
	return append(dst, val...)
}

// AppendNull appends a null value to dst as either a primitive or container
// value and returns the extended buffer.
func AppendNull(dst Bytes) Bytes {
	return AppendUvarint(dst, tagNull)
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

func containerTag(length int) uint64 {
	return (uint64(length) << 1) | 1
}

func primitiveTag(length int) uint64 {
	return (uint64(length) + 1) << 1
}

const tagNull = 0

func tagIsContainer(t uint64) bool {
	return t&1 == 1
}

func tagIsNull(t uint64) bool {
	return t == tagNull
}

func tagLength(t uint64) int {
	return int(t>>1 - (^t & 1))
}
