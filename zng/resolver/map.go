package resolver

import (
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

// Map is a table of descriptors respresented as a golang map.  Map implements
// the zbuf.Resolver interface.
type Map struct {
	table map[int]*zbuf.Descriptor
}

func NewMap() *Map {
	return &Map{
		table: make(map[int]*zbuf.Descriptor),
	}
}

// Enter creates a zbuf.Descriptor for the indicated zng.Type with the
// indicate td, and records the binding in the Map.
func (m *Map) Enter(td int, typ *zng.TypeRecord) *zbuf.Descriptor {
	d := zbuf.NewDescriptor(typ)
	d.ID = td
	m.table[td] = d
	return d
}

// Lookup implements zbuf.Resolver
func (m *Map) Lookup(td int) *zbuf.Descriptor {
	return m.table[td]
}
