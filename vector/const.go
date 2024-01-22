package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Const struct {
	val   zed.Value
	len   uint32
	Nulls *Bool
}

var _ Any = (*Const)(nil)

func NewConst(val zed.Value, len uint32, nulls *Bool) *Const {
	return &Const{val: val, len: len, Nulls: nulls}
}

func (c *Const) Type() zed.Type {
	return c.val.Type()
}

func (c *Const) Len() uint32 {
	return c.len
}

func (*Const) Ref()   {}
func (*Const) Unref() {}

func (c *Const) Length() int {
	return int(c.len)
}

func (c *Const) Value() zed.Value {
	return c.val
}

func (c *Const) Serialize(b *zcode.Builder, slot uint32) {
	if c.Nulls != nil && c.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(c.val.Bytes())
	}
}
