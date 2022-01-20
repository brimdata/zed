package zed

import (
	"github.com/brimdata/zed/zcode"
)

type TypeOfNull struct{}

func (t *TypeOfNull) ID() int {
	return IDNull
}

func (t *TypeOfNull) String() string {
	return "null"
}

func (t *TypeOfNull) Marshal(zcode.Bytes) interface{} {
	return nil
}

func (t *TypeOfNull) Format(zcode.Bytes) string {
	return "null"
}
