package zng

import (
	"encoding/json"
	"strings"
)

// Column defines the field name and type of a column in a record type.
type Column struct {
	Name string
	Type Type
}

func (c *Column) String() string {
	return c.Name + ":" + c.Type.String()
}

type sColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *Column) MarshalJSON() ([]byte, error) {
	col := sColumn{
		Name: c.Name,
		Type: c.Type.String(),
	}
	return json.Marshal(col)
}

func (c *Column) UnmarshalJSON(data []byte) error {
	var v sColumn
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	t, err := LookupType(v.Type)
	if err != nil {
		return err
	}
	c.Name = v.Name
	c.Type = t
	return nil
}

func columnList(prefix string, columns []Column, suffix string) string {
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
