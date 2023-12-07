package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Bool struct {
	mem
	Typ    zed.Type
	Values []bool //XXX bit vector
	Nulls  Nullmask
}

var _ Any = (*Bool)(nil)

func NewBool(typ zed.Type, vals []bool, nulls Nullmask) *Bool {
	return &Bool{Typ: typ, Values: vals, Nulls: nulls}
}

func (b *Bool) Type() zed.Type {
	return b.Typ
}

func (b *Bool) NewBuilder() Builder {
	vals := b.Values
	nulls := b.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if !nulls.Has(uint32(voff)) {
				b.Append(zed.EncodeBool(vals[voff]))
			} else {
				b.Append(nil)
			}
			voff++
			return true

		}
		return false
	}
}
