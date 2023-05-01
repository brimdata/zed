package vector

import (
	"github.com/brimdata/zed"
)

type Array struct {
	mem
	Typ     *zed.TypeArray //XXX type array or set
	Lengths []int32
	Values  Any
}

var _ Any = (*Array)(nil)

func NewArray(typ *zed.TypeArray, lengths []int32, values Any) *Array {
	return &Array{Typ: typ, Lengths: lengths, Values: values}
}

func (a *Array) Type() zed.Type {
	return a.Typ
}
