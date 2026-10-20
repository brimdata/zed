package vector

import (
	"github.com/superdb/super"
	"github.com/superdb/super/scode"
)

// None is a vector of singleton none values.  A none value has no use
// other than being cast to an option type, which yields that option's none.
type None struct {
	len uint32
}

func NewNone(len uint32) *None {
	return &None{len: len}
}

func (*None) Kind() Kind {
	return KindNone
}

func (n *None) Len() uint32 {
	return n.len
}

func (*None) Serialize(b *scode.Builder, _ uint32) {
	b.Append(nil)
}

func (*None) Type() super.Type {
	return super.TypeNone
}
