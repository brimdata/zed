package zed

import (
	"strconv"

	"github.com/brimdata/zed/zcode"
)

func NewUint64(v uint64) *Value {
	return &Value{TypeUint64, EncodeUint(v)}
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

func DecodeInt(zv zcode.Bytes) int64 {
	return zcode.DecodeCountedVarint(zv)
}

func DecodeUint(zv zcode.Bytes) uint64 {
	return zcode.DecodeCountedUvarint(zv)
}

type TypeOfInt8 struct{}

func (t *TypeOfInt8) ID() int {
	return IDInt8
}

func (t *TypeOfInt8) String() string {
	return "int8"
}

func (t *TypeOfInt8) Marshal(zv zcode.Bytes) interface{} {
	return DecodeInt(zv)
}

func (t *TypeOfInt8) Format(zv zcode.Bytes) string {
	return strconv.FormatInt(DecodeInt(zv), 10)
}

type TypeOfUint8 struct{}

func (t *TypeOfUint8) ID() int {
	return IDUint8
}

func (t *TypeOfUint8) String() string {
	return "uint8"
}

func (t *TypeOfUint8) Marshal(zv zcode.Bytes) interface{} {
	return DecodeUint(zv)
}

func (t *TypeOfUint8) Format(zv zcode.Bytes) string {
	return strconv.FormatUint(DecodeUint(zv), 10)
}

type TypeOfInt16 struct{}

func (t *TypeOfInt16) ID() int {
	return IDInt16
}

func (t *TypeOfInt16) String() string {
	return "int16"
}

func (t *TypeOfInt16) Marshal(zv zcode.Bytes) interface{} {
	return DecodeInt(zv)
}

func (t *TypeOfInt16) Format(zv zcode.Bytes) string {
	return strconv.FormatInt(DecodeInt(zv), 10)
}

type TypeOfUint16 struct{}

func (t *TypeOfUint16) ID() int {
	return IDUint16
}

func (t *TypeOfUint16) String() string {
	return "uint16"
}

func (t *TypeOfUint16) Marshal(zv zcode.Bytes) interface{} {
	return DecodeUint(zv)
}

func (t *TypeOfUint16) Format(zv zcode.Bytes) string {
	return strconv.FormatUint(DecodeUint(zv), 10)
}

type TypeOfInt32 struct{}

func (t *TypeOfInt32) ID() int {
	return IDInt32
}

func (t *TypeOfInt32) String() string {
	return "int32"
}

func (t *TypeOfInt32) Marshal(zv zcode.Bytes) interface{} {
	return DecodeInt(zv)
}

func (t *TypeOfInt32) Format(zv zcode.Bytes) string {
	return strconv.FormatInt(DecodeInt(zv), 10)
}

type TypeOfUint32 struct{}

func (t *TypeOfUint32) ID() int {
	return IDUint32
}

func (t *TypeOfUint32) String() string {
	return "uint32"
}

func (t *TypeOfUint32) Marshal(zv zcode.Bytes) interface{} {
	return DecodeUint(zv)
}

func (t *TypeOfUint32) Format(zv zcode.Bytes) string {
	return strconv.FormatUint(DecodeUint(zv), 10)
}

type TypeOfInt64 struct{}

func (t *TypeOfInt64) ID() int {
	return IDInt64
}

func (t *TypeOfInt64) String() string {
	return "int64"
}

func (t *TypeOfInt64) Marshal(zv zcode.Bytes) interface{} {
	return DecodeInt(zv)
}

func (t *TypeOfInt64) Format(zv zcode.Bytes) string {
	return strconv.FormatInt(DecodeInt(zv), 10)
}

type TypeOfUint64 struct{}

func (t *TypeOfUint64) ID() int {
	return IDUint64
}

func (t *TypeOfUint64) String() string {
	return "uint64"
}

func (t *TypeOfUint64) Marshal(zv zcode.Bytes) interface{} {
	return DecodeUint(zv)
}

func (t *TypeOfUint64) Format(zv zcode.Bytes) string {
	return strconv.FormatUint(DecodeUint(zv), 10)
}
