package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Array struct {
	Typ     *zed.TypeArray
	Offsets []uint32
	Values  Any
	Nulls   *Bool
}

var _ Any = (*Array)(nil)

func NewArray(typ *zed.TypeArray, offsets []uint32, values Any, nulls *Bool) *Array {
	return &Array{Typ: typ, Offsets: offsets, Values: values, Nulls: nulls}
}

func (a *Array) Type() zed.Type {
	return a.Typ
}

func (a *Array) Len() uint32 {
	return uint32(len(a.Offsets) - 1)
}

func (a *Array) Serialize(b *zcode.Builder, slot uint32) {
	if a.Nulls != nil && a.Nulls.Value(slot) {
		b.Append(nil)
		return
	}
	off := a.Offsets[slot]
	b.BeginContainer()
	for end := a.Offsets[slot+1]; off < end; off++ {
		a.Values.Serialize(b, off)
	}
	b.EndContainer()
}
