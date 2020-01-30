package zng

import (
	"github.com/mccanne/zq/zcode"
)

type TypeOfEnum struct{}

func NewEnum(s string) Value {
	return Value{TypeEnum, EncodeEnum(s)}
}

func EncodeEnum(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeEnum(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfEnum) Parse(in []byte) (zcode.Bytes, error) {
	return in, nil
}

func (t *TypeOfEnum) ID() int {
	return IdEnum
}

func (t *TypeOfEnum) String() string {
	return "enum"
}

func (t *TypeOfEnum) StringOf(zv zcode.Bytes, _ OutFmt) string {
	e, err := DecodeEnum(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return e
}

func (t *TypeOfEnum) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeEnum(zv)
}
