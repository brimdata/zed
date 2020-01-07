package resolver

import "github.com/mccanne/zq/zbuf"

// Slice is a table of descriptors respresented as a slice and grown
// on demand as small-in type descriptors are entered into the table.
type Slice struct {
	table []*zbuf.Descriptor
}

func (s *Slice) lookup(td int) *zbuf.Descriptor {
	if td >= 0 && td < len(s.table) {
		return s.table[td]
	}
	return nil
}

func (s *Slice) enter(td int, d *zbuf.Descriptor) {
	if td >= len(s.table) {
		new := make([]*zbuf.Descriptor, td+1)
		copy(new, s.table)
		s.table = new
	}
	s.table[td] = d
}
