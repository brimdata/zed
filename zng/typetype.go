package zng

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
)

type TypeOfType struct{}

func NewTypeType(t Type) Value {
	return Value{TypeType, zcode.Bytes(t.ZSON())}
}

func (t *TypeOfType) ID() int {
	return IdType
}

func (t *TypeOfType) String() string {
	return "type"
}

func (t *TypeOfType) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.ZSONOf(zv), nil
}

func (t *TypeOfType) ZSON() string {
	return "type"
}

func (t *TypeOfType) ZSONOf(zv zcode.Bytes) string {
	return fmt.Sprintf("(%s)", string(zv))
}
