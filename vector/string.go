package vector

import (
	"github.com/brimdata/zed"
)

type String struct {
	mem
	Typ    zed.Type
	Values []string
	Nulls  Nullmask
}

var _ Any = (*String)(nil)

func NewString(typ zed.Type, vals []string, nulls Nullmask) *String {
	return &String{Typ: typ, Values: vals, Nulls: nulls}
}

func (s *String) Type() zed.Type {
	return s.Typ
}
