package vector

import (
	"github.com/brimdata/zed"
)

type Bool struct {
	mem
	Typ    zed.Type
	Values []bool //XXX bit vector
	Nulls  Nullmask
}

var _ Any = (*Int)(nil)

func NewBool(typ zed.Type, vals []bool, nulls Nullmask) *Bool {
	return &Bool{Typ: typ, Values: vals, Nulls: nulls}
}

func (b *Bool) Type() zed.Type {
	return b.Typ
}
