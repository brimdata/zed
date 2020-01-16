package zng

import (
	"errors"
	"strconv"

	"github.com/mccanne/zq/zcode"
)

type TypeOfPort struct{}

func NewPort(p uint32) Value {
	return Value{TypePort, EncodePort(p)}
}

func EncodePort(p uint32) zcode.Bytes {
	var b [2]byte
	b[0] = byte(p >> 8)
	b[1] = byte(p)
	return b[:]
}

func DecodePort(zv zcode.Bytes) (uint32, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	if len(zv) != 2 {
		return 0, errors.New("port encoding must be 2 bytes")

	}
	return uint32(zv[0])<<8 | uint32(zv[1]), nil
}

func (t *TypeOfPort) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseUint32(in)
	if err != nil {
		return nil, err
	}
	return EncodePort(i), nil
}

func (t *TypeOfPort) String() string {
	return "port"
}

func (t *TypeOfPort) StringOf(zv zcode.Bytes) string {
	p, err := DecodePort(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatUint(uint64(p), 10)
}

func (t *TypeOfPort) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodePort(zv)
}
