package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
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
	bytes := c.val.Bytes()
	var voff uint32
	return func(b *zcode.Builder) bool {
		if voff >= c.len {
			return false
		}
		b.Append(bytes)
		voff++
		return true
	}
}

func (c *Const) Length() int {
	return int(c.len)
}

func (c *Const) Value() *zed.Value {
	return c.val
}

// XXX should Const wrap a single-element vector?
func (c *Const) Key(b []byte, slot int) []byte {
	return append(b, c.val.Bytes()...)
}

func (c *Const) Serialize(slot int) *zed.Value {
	return c.val
}
