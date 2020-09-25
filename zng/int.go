package zng

import (
	"strconv"

	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/zcode"
)

func NewUint64(v uint64) Value {
	return Value{TypeUint64, EncodeUint(v)}
}

func EncodeInt(i int64) zcode.Bytes {
	var b [8]byte
	n := zcode.EncodeCountedVarint(b[:], i)
	return b[:n]
}

func AppendInt(bytes zcode.Bytes, i int64) zcode.Bytes {
	return zcode.AppendCountedVarint(bytes, i)
}

func EncodeUint(i uint64) zcode.Bytes {
	var b [8]byte
	n := zcode.EncodeCountedUvarint(b[:], i)
	return b[:n]
}

func AppendUint(bytes zcode.Bytes, i uint64) zcode.Bytes {
	return zcode.AppendCountedUvarint(bytes, i)
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

type TypeOfInt8 struct{}

func (t *TypeOfInt8) ID() int {
	return IdInt8
}

func (t *TypeOfInt8) String() string {
	return "int8"
}

func (t *TypeOfInt8) Parse(in []byte) (zcode.Bytes, error) {
	b, err := byteconv.ParseInt8(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(int64(b)), nil
}

func (t *TypeOfInt8) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	b, err := DecodeInt(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatInt(int64(b), 10)
}

func (t *TypeOfInt8) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeInt(zv)
}

type TypeOfUint8 struct{}

func (t *TypeOfUint8) ID() int {
	return IdUint8
}

func (t *TypeOfUint8) String() string {
	return "uint8"
}

func (t *TypeOfUint8) Parse(in []byte) (zcode.Bytes, error) {
	b, err := byteconv.ParseUint8(in)
	if err != nil {
		return nil, err
	}
	return EncodeUint(uint64(b)), nil
}

func (t *TypeOfUint8) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	b, err := DecodeUint(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatUint(uint64(b), 10)
}

func (t *TypeOfUint8) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeUint(zv)
}

type TypeOfInt16 struct{}

func (t *TypeOfInt16) ID() int {
	return IdInt16
}

func (t *TypeOfInt16) String() string {
	return "int16"
}

func (t *TypeOfInt16) Parse(in []byte) (zcode.Bytes, error) {
	i, err := byteconv.ParseInt16(in)
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
	i, err := byteconv.ParseUint16(in)
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
	i, err := byteconv.ParseInt32(in)
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
	i, err := byteconv.ParseUint32(in)
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
	i, err := byteconv.ParseInt64(in)
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
	i, err := byteconv.ParseUint64(in)
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
