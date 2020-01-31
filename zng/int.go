package zng

import (
	"strconv"

	"github.com/mccanne/zq/zcode"
)

type TypeOfInt struct{}

func NewInt(i int64) Value {
	return Value{TypeInt, EncodeInt(i)}
}

func EncodeInt(i int64) zcode.Bytes {
	var b [8]byte
	n := zcode.EncodeCountedVarint(b[:], i)
	return b[:n]
}

func DecodeInt(zv zcode.Bytes) (int64, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return zcode.DecodeCountedVarint(zv), nil
}

func (t *TypeOfInt) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseInt64(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(i), nil
}

func (t *TypeOfInt) ID() int {
	return IdInt64
}

func (t *TypeOfInt) String() string {
	return "int"
}

func (t *TypeOfInt) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	i, err := DecodeInt(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatInt(i, 10)
}

func (t *TypeOfInt) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeInt(zv)
}
