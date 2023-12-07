package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Uint struct {
	mem
	Typ    zed.Type
	Values []uint64
	Nulls  Nullmask
}

var _ Any = (*Uint)(nil)

func NewUint(typ zed.Type, vals []uint64, nulls Nullmask) *Uint {
	return &Uint{Typ: typ, Values: vals, Nulls: nulls}
}

func (u *Uint) Type() zed.Type {
	return u.Typ
}

func (u *Uint) NewBuilder() Builder {
	vals := u.Values
	nulls := u.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if !nulls.Has(uint32(voff)) {
				b.Append(zed.EncodeUint(vals[voff]))
			} else {
				b.Append(nil)
			}
			voff++
			return true

		}
		return false
	}
}
