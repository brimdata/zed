package vector

import (
	"github.com/brimdata/zed"
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
