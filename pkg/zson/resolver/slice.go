package resolver

import "github.com/mccanne/zq/pkg/zson"

// Slice is a table of descriptors respresented as a slice and grown
// on demand as small-in type descriptors are entered into the table.
type Slice struct {
	table []*zson.Descriptor
}

func (s *Slice) lookup(td int) *zson.Descriptor {
	if td >= 0 && td < len(s.table) {
		return s.table[td]
	}
	return nil
}

func (s *Slice) enter(td int, d *zson.Descriptor) {
	if td >= len(s.table) {
		new := make([]*zson.Descriptor, td+1)
		copy(new, s.table)
		s.table = new
	}
	s.table[td] = d
}
