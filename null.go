package zed

import (
	"github.com/brimdata/zed/zcode"
)

var Null = &Value{Type: TypeNull}

type TypeOfNull struct{}

func (t *TypeOfNull) ID() int {
	return IDNull
}

func (t *TypeOfNull) String() string {
	return "null"
}

func (t *TypeOfNull) Marshal(zv zcode.Bytes) (interface{}, error) {
	return nil, nil
}

func (t *TypeOfNull) Format(zv zcode.Bytes) string {
	return "null"
}
