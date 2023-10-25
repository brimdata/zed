package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Int struct {
	mem
	Typ    zed.Type
	Values []int64
	Nulls  Nullmask
}

var _ Any = (*Int)(nil)

func NewInt(typ zed.Type, vals []int64, nulls Nullmask) *Int {
	return &Int{Typ: typ, Values: vals, Nulls: nulls}
}

func (i *Int) Type() zed.Type {
	return i.Typ
}

func (i *Int) NewBuilder() Builder {
	vals := i.Values
	nulls := i.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if nulls.Has(uint32(voff)) {
				b.Append(nil)
			} else {
				b.Append(zed.EncodeInt(vals[voff]))
			}
			voff++
			return true

		}
		return false
	}
}
