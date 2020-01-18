package zng

import (
	"encoding/json"
	"strings"
)

type ColumnProto struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Column defines the field name and type of a column in a record type.
type Column struct {
	ColumnProto
	Type Type
}

func NewColumn(name string, typ Type) Column {
	c := Column{}
	c.Name = name
	c.Type = typ
	return c
}

func (c *Column) String() string {
	return c.Name + ":" + c.Type.String()
}

func (c *Column) MarshalJSON() ([]byte, error) {
	if c.ColumnProto.Type == "" {
		c.ColumnProto.Type = c.Type.String()
	}
	return json.Marshal(c.ColumnProto)
}

// Typify fills in the native Type value for each column given a type context.
// If the context  is nil, then types are looked up as primitive types.
func Typify(c Context, columns []Column) error {
	for k := range columns {
		if columns[k].Type == nil {
			typeName := columns[k].ColumnProto.Type
			var typ Type
			if c != nil {
				var err error
				typ, err = c.LookupByName(typeName)
				if err != nil {
					return err
				}
			} else {
				typ = LookupPrimitive(typeName)
			}
			columns[k].Type = typ
		}
	}
	return nil
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
