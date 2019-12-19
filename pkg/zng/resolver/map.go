package resolver

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
)

// Map is a table of descriptors respresented as a golang map.  Map implements
// the zng.Resolver interface.
type Map struct {
	table map[int]*zng.Descriptor
}

func NewMap() *Map {
	return &Map{
		table: make(map[int]*zng.Descriptor),
	}
}

// Enter creates a zng.Descriptor for the indicated zeek.Type with the
// indicate td, and records the binding in the Map.
func (m *Map) Enter(td int, typ *zeek.TypeRecord) *zng.Descriptor {
	d := zng.NewDescriptor(typ)
	d.ID = td
	m.table[td] = d
	return d
}

// Lookup implements zng.Resolver
func (m *Map) Lookup(td int) *zng.Descriptor {
	return m.table[td]
}
