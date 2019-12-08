package resolver

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
)

// Map is a table of descriptors respresented as a golang map.  Map implements
// the zq.Resolver interface.
type Map struct {
	table map[int]*zq.Descriptor
}

func NewMap() *Map {
	return &Map{
		table: make(map[int]*zq.Descriptor),
	}
}

// Enter creates a zq.Descriptor for the indicated zeek.Type with the
// indicate td, and records the binding in the Map.
func (m *Map) Enter(td int, typ *zeek.TypeRecord) *zq.Descriptor {
	d := zq.NewDescriptor(typ)
	d.ID = td
	m.table[td] = d
	return d
}

// Lookup implements zq.Resolver
func (m *Map) Lookup(td int) *zq.Descriptor {
	return m.table[td]
}
