package vector

import (
	"github.com/brimdata/zed"
)

type Const struct {
	val *zed.Value
	len uint32
}

func NewConst(val *zed.Value, len uint32) *Const {
	return &Const{val: val, len: len}
}

func (c *Const) Type() zed.Type {
	return c.val.Type
}

func (*Const) Ref()   {}
func (*Const) Unref() {}

func (c *Const) NewBuilder() Builder {
	return nil //XXX
}

func (c *Const) Length() int {
	return int(c.len)
}

func (c *Const) Value() *zed.Value {
	return c.val
}
