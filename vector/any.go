package vector

import (
	"github.com/brimdata/zed"
)

type Any interface {
	Type() zed.Type
	Ref()
	Unref()
}

// XXX move to vector
/*
func Under(a Any) Any {
	for {
		if nulls, ok := a.(*Nulls); ok {
			a = nulls.values
			continue
		}
		return a
	}
}
*/

type Const struct {
	val *zed.Value
}

func NewConst(val *zed.Value) *Const {
	return &Const{val: val}
}

func (c *Const) Type() zed.Type {
	return c.val.Type
}

func (*Const) Ref()   {}
func (*Const) Unref() {}
