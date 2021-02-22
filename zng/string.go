package zng

import (
	"github.com/brimsec/zq/zcode"
)

type TypeOfString struct{}

func NewString(s string) Value {
	return Value{TypeString, EncodeString(s)}
}

func EncodeString(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeString(zv zcode.Bytes) (string, error) {
	return string(zv), nil
}

func (t *TypeOfString) ID() int {
	return IdString
}

func (t *TypeOfString) String() string {
	return "string"
}

func (t *TypeOfString) Marshal(zv zcode.Bytes) (interface{}, error) {
	return string(zv), nil
}

func (t *TypeOfString) ZSON() string {
	return "string"
}

func (t *TypeOfString) ZSONOf(zv zcode.Bytes) string {
	return QuotedString(zv, false)
}
