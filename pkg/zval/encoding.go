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
	"errors"
)

var (
	ErrNotContainer = errors.New("not a container")
	ErrNotSingleton = errors.New("not a single container")
)

// Encoding is the serialized representation of zson values.
type Encoding []byte

func (e Encoding) Bytes() []byte {
	return []byte(e)
}

func (e Encoding) Iter() Iter {
	return Iter(e)
}

func (e Encoding) String() string {
	b, err := e.Build(nil)
	if err != nil {
		panic("zval encoding has bad format: " + err.Error())
	}
	return string(b)
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

func (e Encoding) Build(b []byte) ([]byte, error) {
	for it := Iter(e); !it.Done(); {
		v, container, err := it.Next()
		if err != nil {
			return nil, err
		}
		if container {
			if v == nil {
				b = append(b, '(')
				b = append(b, '*')
				b = append(b, ')')
				continue
			}
			b = append(b, '[')
			b, err = v.Build(b)
			if err != nil {
				return nil, err
			}
			b = append(b, ']')
		} else {
			b = append(b, '(')
			b = appendBytes(b, v)
			b = append(b, ')')
		}
	}
	return b, nil
}

// Body returns the contents of an encoding that represents a container as
// an encoding of the list of values.  If the encoding is not a container,
// ErrNotContainer is returned.  If the encoding is not a single container,
// ErrNotSingleton is returned.
func (e Encoding) Body() (Encoding, error) {
	it := Iter(e)
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

// AppendValue encodes each byte slice as a value Encoding, concatenates the
// values as an aggregate, then encodes the aggregate as a container Encoding.
func AppendContainer(dst Encoding, vals [][]byte) Encoding {
	if vals == nil {
		return AppendUvarint(dst, containerTagUnset)
	}
	var n int
	for _, v := range vals {
		n += sizeOfValue(len(v))
	}
	dst = AppendUvarint(dst, containerTag(n))
	for _, v := range vals {
		dst = AppendValue(dst, v)
	}
	return dst
}

// AppendContainerValue takes an Encoding that is encoded as a list of Encodings
// and concatenates it as a container Encoding.
func AppendContainerValue(dst Encoding, val Encoding) Encoding {
	if val == nil {
		return AppendUvarint(dst, containerTagUnset)
	}
	dst = AppendUvarint(dst, containerTag(len(val)))
	dst = append(dst, val...)
	return dst
}

// AppendValue encodes the byte slice as value Encoding, appends it
// to dst, and returns appended Encoding.
func AppendValue(dst Encoding, val []byte) Encoding {
	if val == nil {
		return AppendUvarint(dst, valueTagUnset)
	}
	dst = AppendUvarint(dst, valueTag(len(val)))
	return append(dst, val...)
}

func Append(dst Encoding, val []byte, container bool) Encoding {
	if container {
		return AppendContainerValue(dst, val)
	}
	return AppendValue(dst, val)
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
