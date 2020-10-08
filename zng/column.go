package zng

import (
	"strings"
)

// Column defines the field name and type of a column in a record type.
type Column struct {
	Name string
	Type Type
}

func NewColumn(name string, typ Type) Column {
	return Column{name, typ}
}

func (c *Column) String() string {
	return FormatName(c.Name) + ":" + c.Type.String()
}

func ColumnString(prefix string, columns []Column, suffix string) string {
	var s strings.Builder
	s.WriteString(prefix)
	var comma bool
	for _, c := range columns {
		if comma {
			s.WriteByte(byte(','))
		}
		s.WriteString(c.String())
		comma = true
	}
	s.WriteString(suffix)
	return s.String()
}
