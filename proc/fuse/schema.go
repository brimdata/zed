package fuse

import (
	"fmt"

	"github.com/brimsec/zq/zng"
)

type schema struct {
	columns []zng.Column
	// keyed on name + type ID
	position map[string]int
	// keyed on name
	count map[string]int
}

func newSchema() *schema {
	return &schema{
		position: make(map[string]int),
		count:    make(map[string]int),
	}
}

func (s *schema) touch(name string) int {
	cnt := s.count[name] + 1
	s.count[name] = cnt
	return cnt
}

func (s *schema) mixin(typ *zng.TypeRecord) []int {
	var positions []int
	for _, c := range typ.Columns {
		name := c.Name
		key := fmt.Sprintf("%s%d", name, c.Type.ID())
		uberPosition, ok := s.position[key]
		if !ok {
			cnt := s.touch(name)
			if cnt > 1 {
				name = fmt.Sprintf("%s_%d", name, cnt)
			}
			uberPosition = len(s.columns)
			s.position[key] = uberPosition
			s.columns = append(s.columns, zng.Column{name, c.Type})
		}
		positions = append(positions, uberPosition)
	}
	return positions
}
