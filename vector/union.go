package vector

import (
	"github.com/brimdata/zed"
)

type Union struct {
	mem
	Typ    *zed.TypeUnion
	Values []Any
}

var _ Any = (*Union)(nil)

func NewUnion(typ *zed.TypeUnion) *Union {
	return &Union{Typ: typ, Values: make([]Any, len(typ.Types))}
}

func (u *Union) Type() zed.Type {
	return u.Typ
}
