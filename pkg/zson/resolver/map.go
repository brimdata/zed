package resolver

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

// Map is a table of descriptors respresented as a golang map.  Map implements
// the zson.Resolver interface.
type Map struct {
	table map[int]*zson.Descriptor
}

func NewMap() *Map {
	return &Map{
		table: make(map[int]*zson.Descriptor),
	}
}

// Enter creates a zson.Descriptor for the indicated zeek.Type with the
// indicate td, and records the binding in the Map.
func (m *Map) Enter(td int, typ *zeek.TypeRecord) *zson.Descriptor {
	d := zson.NewDescriptor(typ)
	d.ID = td
	m.table[td] = d
	return d
}

// Lookup implements zson.Resolver
func (m *Map) Lookup(td int) *zson.Descriptor {
	return m.table[td]
}
