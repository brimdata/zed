package resolver

import (
	"github.com/mccanne/zq/zng"
)

// Map is a table of descriptors respresented as a golang map.  Map implements
// the zbuf.Resolver interface.
type Map struct {
	table map[int]*zng.TypeRecord
}

func NewMap() *Map {
	return &Map{
		table: make(map[int]*zng.TypeRecord),
	}
}

// Enter creates a zng.TypeRecord for the indicated zng.Type with the
// indicate td, and records the binding in the Map.
func (m *Map) Enter(td int, typ *zng.TypeRecord) *zng.TypeRecord {
	m.table[td] = zng.CopyTypeRecord(td, typ)
	return typ
}

// Lookup implements zng.TypeRecord
func (m *Map) Lookup(td int) *zng.TypeRecord {
	return m.table[td]
}
