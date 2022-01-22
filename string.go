package zed

import (
	"github.com/brimdata/zed/zcode"
)

type TypeOfString struct{}

func NewString(s string) *Value {
	return &Value{TypeString, EncodeString(s)}
}

func EncodeString(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeString(zv zcode.Bytes) string {
	return string(zv)
}

func (t *TypeOfString) ID() int {
	return IDString
}

func (t *TypeOfString) Kind() string {
	return "primitive"
}
