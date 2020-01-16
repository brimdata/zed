package zng

import (
	"strconv"

	"github.com/mccanne/zq/zcode"
)

type TypeOfCount struct{}

func NewCount(c uint64) Value {
	return Value{TypeCount, EncodeCount(c)}
}

func EncodeCount(c uint64) zcode.Bytes {
	var b [8]byte
	n := encodeUint(b[:], c)
	return b[:n]
}

func DecodeCount(zv zcode.Bytes) (uint64, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return uint64(decodeUint(zv)), nil
}

func (t *TypeOfCount) Parse(in []byte) (zcode.Bytes, error) {
	c, err := UnsafeParseUint64(in)
	if err != nil {
		return nil, err
	}
	return EncodeCount(c), nil
}

func (t *TypeOfCount) String() string {
	return "count"
}

func (t *TypeOfCount) StringOf(zv zcode.Bytes) string {
	c, err := DecodeCount(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatUint(c, 10)
}

func (t *TypeOfCount) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeCount(zv)
}
