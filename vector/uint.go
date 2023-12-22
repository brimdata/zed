package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Uint struct {
	mem
	Typ    zed.Type
	Values []uint64
}

var _ Any = (*Uint)(nil)

func NewUint(typ zed.Type, values []uint64) *Uint {
	return &Uint{Typ: typ, Values: values}
}

func (u *Uint) Type() zed.Type {
	return u.Typ
}

func (u *Uint) NewBuilder() Builder {
	vals := u.Values
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			b.Append(zed.EncodeUint(vals[voff]))
			voff++
			return true
		}
		return false
	}
}
