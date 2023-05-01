package vector

import (
	"github.com/brimdata/zed"
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
