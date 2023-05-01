package vector

import (
	"github.com/brimdata/zed"
)

type Map struct {
	mem
	Typ    *zed.TypeMap
	Keys   Any
	Values Any
}

var _ Any = (*Map)(nil)

func NewMap(typ *zed.TypeMap, keys Any, values Any) *Map {
	return &Map{Typ: typ, Keys: keys, Values: values}
}

func (m *Map) Type() zed.Type {
	return m.Typ
}
