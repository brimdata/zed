package zng

import (
	"strconv"

	"github.com/brimsec/zq/zcode"
)

func NewUint64(v uint64) Value {
	return Value{TypeUint64, EncodeUint(v)}
}

func EncodeByte(b byte) zcode.Bytes {
	return []byte{b}
}

func EncodeInt(i int64) zcode.Bytes {
	var b [8]byte
	n := zcode.EncodeCountedVarint(b[:], i)
	return b[:n]
}

func EncodeUint(i uint64) zcode.Bytes {
	var b [8]byte
	n := zcode.EncodeCountedUvarint(b[:], i)
	return b[:n]
}

func DecodeByte(zv zcode.Bytes) (byte, error) {
	if len(zv) != 1 {
		return 0, ErrUnset
	}
	return zv[0], nil
}

func DecodeInt(zv zcode.Bytes) (int64, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return zcode.DecodeCountedVarint(zv), nil
}

func DecodeUint(zv zcode.Bytes) (uint64, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return zcode.DecodeCountedUvarint(zv), nil
}

func stringOfInt(zv zcode.Bytes, t Type) string {
	i, err := DecodeInt(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatInt(i, 10)
}

func stringOfUint(zv zcode.Bytes, t Type) string {
	i, err := DecodeUint(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatUint(i, 10)
}

type TypeOfByte struct{}

func (t *TypeOfByte) ID() int {
	return IdByte
}

func (t *TypeOfByte) String() string {
	return "byte"
}

func (t *TypeOfByte) Parse(in []byte) (zcode.Bytes, error) {
	b, err := UnsafeParseUint8(in)
	if err != nil {
		return nil, err
	}
	return EncodeByte(b), nil
}

func (t *TypeOfByte) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	b, err := DecodeByte(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatUint(uint64(b), 10)
}

func (t *TypeOfByte) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeByte(zv)
}

type TypeOfInt16 struct{}

func (t *TypeOfInt16) ID() int {
	return IdInt16
}

func (t *TypeOfInt16) String() string {
	return "int16"
}

func (t *TypeOfInt16) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseInt16(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(int64(i)), nil
}

func (t *TypeOfInt16) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return stringOfInt(zv, t)
}

func (t *TypeOfInt16) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeInt(zv)
}

type TypeOfUint16 struct{}

func (t *TypeOfUint16) ID() int {
	return IdUint16
}

func (t *TypeOfUint16) String() string {
	return "uint16"
}

func (t *TypeOfUint16) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseUint16(in)
	if err != nil {
		return nil, err
	}
	return EncodeUint(uint64(i)), nil
}

func (t *TypeOfUint16) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return stringOfUint(zv, t)
}

func (t *TypeOfUint16) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeUint(zv)
}

type TypeOfInt32 struct{}

func (t *TypeOfInt32) ID() int {
	return IdInt32
}

func (t *TypeOfInt32) String() string {
	return "int32"
}

func (t *TypeOfInt32) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseInt32(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(int64(i)), nil
}

func (t *TypeOfInt32) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return stringOfInt(zv, t)
}

func (t *TypeOfInt32) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeInt(zv)
}

type TypeOfUint32 struct{}

func (t *TypeOfUint32) ID() int {
	return IdUint32
}

func (t *TypeOfUint32) String() string {
	return "uint32"
}

func (t *TypeOfUint32) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseUint32(in)
	if err != nil {
		return nil, err
	}
	return EncodeUint(uint64(i)), nil
}

func (t *TypeOfUint32) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return stringOfUint(zv, t)
}

func (t *TypeOfUint32) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeUint(zv)
}

type TypeOfInt64 struct{}

func (t *TypeOfInt64) ID() int {
	return IdInt64
}

func (t *TypeOfInt64) String() string {
	return "int64"
}

func (t *TypeOfInt64) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseInt64(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(int64(i)), nil
}

func (t *TypeOfInt64) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return stringOfInt(zv, t)
}

func (t *TypeOfInt64) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeInt(zv)
}

type TypeOfUint64 struct{}

func (t *TypeOfUint64) ID() int {
	return IdUint64
}

func (t *TypeOfUint64) String() string {
	return "uint64"
}

func (t *TypeOfUint64) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseUint64(in)
	if err != nil {
		return nil, err
	}
	return EncodeUint(uint64(i)), nil
}

func (t *TypeOfUint64) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return stringOfUint(zv, t)
}

func (t *TypeOfUint64) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeUint(zv)
}
