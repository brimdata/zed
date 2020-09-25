package zng

import (
	"strconv"

	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/zcode"
)

type TypeOfPort struct{}

func NewPort(p uint32) Value {
	return Value{TypePort, EncodeUint(uint64(p))}
}

func (t *TypeOfPort) Parse(in []byte) (zcode.Bytes, error) {
	i, err := byteconv.ParseUint64(in)
	if err != nil {
		return nil, err
	}
	return EncodeUint(i), nil
}

func (t *TypeOfPort) ID() int {
	return IdPort
}

func (t *TypeOfPort) String() string {
	return "port"
}

func (t *TypeOfPort) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	p, err := DecodeUint(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatUint(p, 10)
}

func (t *TypeOfPort) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeUint(zv)
}
