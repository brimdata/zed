package resolver

import (
	"github.com/brimdata/zq/zng"
)

// Slice is a table of descriptors respresented as a slice and grown
// on demand as small-int type descriptors are entered into the table.
type Slice []zng.Type

func (s Slice) Lookup(td int) zng.Type {
	if td >= 0 && td < len(s) {
		return s[td]
	}
	return nil
}

func (s *Slice) Enter(td int, d zng.Type) {
	if td >= len(*s) {
		new := make([]zng.Type, td+1)
		copy(new, *s)
		*s = new
	}
	(*s)[td] = d
}
