package zed

import (
	"github.com/brimdata/zed/zcode"
)

type TypeOfBool struct{}

var False = &Value{TypeBool, []byte{0}}
var True = &Value{TypeBool, []byte{1}}

func IsTrue(zv zcode.Bytes) bool {
	return zv[0] != 0
}

// Not returns the inverse Value of the Boolean-typed bytes value of zb.
func Not(zb zcode.Bytes) *Value {
	if IsTrue(zb) {
		return False
	}
	return True
}

func NewBool(b bool) *Value {
	return &Value{TypeBool, EncodeBool(b)}
}

func AppendBool(zb zcode.Bytes, b bool) zcode.Bytes {
	if b {
		return append(zb, 1)
	}
	return append(zb, 0)
}

func EncodeBool(b bool) zcode.Bytes {
	return AppendBool(nil, b)
}

func DecodeBool(zv zcode.Bytes) bool {
	return zv != nil && zv[0] != 0
}

func (t *TypeOfBool) ID() int {
	return IDBool
}

func (t *TypeOfBool) String() string {
	return "bool"
}

func (t *TypeOfBool) Marshal(zv zcode.Bytes) interface{} {
	return DecodeBool(zv)
}

func (t *TypeOfBool) Format(zv zcode.Bytes) string {
	if DecodeBool(zv) {
		return "true"
	}
	return "false"
}
