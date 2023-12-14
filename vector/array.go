package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Array struct {
	mem
	Typ     zed.Type // Either *zed.TypeArray or *zed.TypeSet.
	Lengths []int32
	Values  Any
}

var _ Any = (*Array)(nil)

func NewArray(typ zed.Type, lengths []int32, values Any) *Array {
	return &Array{Typ: typ, Lengths: lengths, Values: values}
}

func (a *Array) Type() zed.Type {
	return a.Typ
}

func (a *Array) NewBuilder() Builder {
	_, set := zed.TypeUnder(a.Typ).(*zed.TypeSet)
	valueBuilder := a.Values.NewBuilder()
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(a.Lengths) {
			return false
		}
		b.BeginContainer()
		for i := 0; i < int(a.Lengths[off]); i++ {
			if !valueBuilder(b) {
				panic(off)
			}
		}
		if set {
			b.TransformContainer(zed.NormalizeSet)
		}
		b.EndContainer()
		off++
		return true
	}
}

func (a *Array) Key([]byte, int) []byte {
	panic("TBD")
}

func (a *Array) Length() int {
	panic("TBD")
}

func (a *Array) Serialize(int) *zed.Value {
	panic("TBD")
}
