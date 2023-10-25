package vector

import (
	"github.com/brimdata/zed"
)

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

func (c *Const) NewBuilder() Builder {
	return nil //XXX
}
