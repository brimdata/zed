package resolver

import (
	"github.com/brimsec/zq/zng"
)

// Slice is a table of descriptors respresented as a slice and grown
// on demand as small-in type descriptors are entered into the table.
type Slice []*zng.TypeRecord

func (s Slice) Lookup(td int) *zng.TypeRecord {
	if td >= 0 && td < len(s) {
		return s[td]
	}
	return nil
}

func (s *Slice) Enter(td int, d *zng.TypeRecord) {
	if td >= len(*s) {
		new := make([]*zng.TypeRecord, td+1)
		copy(new, *s)
		*s = new
	}
	(*s)[td] = d
}
