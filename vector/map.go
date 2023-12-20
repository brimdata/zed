package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Map struct {
	mem
	Typ     *zed.TypeMap
	Lengths []int32
	Keys    Any
	Values  Any
}

var _ Any = (*Map)(nil)

func NewMap(typ *zed.TypeMap, lengths []int32, keys Any, values Any) *Map {
	return &Map{Typ: typ, Lengths: lengths, Keys: keys, Values: values}
}

func (m *Map) Type() zed.Type {
	return m.Typ
}

func (m *Map) NewBuilder() Builder {
	keyBuilder := m.Keys.NewBuilder()
	valueBuilder := m.Values.NewBuilder()
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(m.Lengths) {
			return false
		}
		b.BeginContainer()
		for i := 0; i < int(m.Lengths[off]); i++ {
			if !keyBuilder(b) {
				panic(off)
			}
			if !valueBuilder(b) {
				panic(off)
			}
		}
		b.TransformContainer(zed.NormalizeMap)
		b.EndContainer()
		off++
		return true
	}

}
