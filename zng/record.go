package zng

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mccanne/zq/zcode"
)

var ErrColumnMismatch = errors.New("zng record mismatch between columns in type and columns in value")

type TypeRecord struct {
	Columns []Column
	Key     string
}

func recordString(columns []Column) string {
	return columnList("record[", columns, "]")
}

func (t *TypeRecord) String() string {
	return recordString(t.Columns)
}

func (t TypeRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Columns)
}

func (t *TypeRecord) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &t.Columns)
}

func parseColumn(in string) (string, Column, error) {
	in = strings.TrimSpace(in)
	colon := strings.IndexByte(in, byte(':'))
	if colon < 0 {
		return "", Column{}, ErrTypeSyntax
	}
	//XXX should check if name is valid
	name := strings.TrimSpace(in[:colon])
	rest, typ, err := parseType(in[colon+1:])
	if err != nil {
		return "", Column{}, err
	}
	if typ == nil {
		return "", Column{}, ErrTypeSyntax
	}
	return rest, Column{Name: name, Type: typ}, nil
}

func match(in, pattern string) (string, bool) {
	in = strings.TrimSpace(in)
	if strings.HasPrefix(in, pattern) {
		return in[len(pattern):], true
	}
	return in, false
}

// parseRecordTypeBody parses a list of record columns of the form "[field:type,...]".
func parseRecordTypeBody(in string) (string, Type, error) {
	in, ok := match(in, "[")
	if !ok {
		return "", nil, ErrTypeSyntax
	}
	var columns []Column
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, col, err := parseColumn(in)
		if err != nil {
			return "", nil, err
		}
		for _, c := range columns {
			if col.Name == c.Name {
				return "", nil, ErrDuplicateFields
			}
		}
		columns = append(columns, col)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if ok {
			return rest, LookupTypeRecord(columns), nil
		}
		return "", nil, ErrTypeSyntax
	}
}

//XXX we shouldn't need this... tests are using it
func (t *TypeRecord) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	var vals []Value
	for i, it := 0, zcode.Iter(zv); !it.Done(); i++ {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		if i >= len(t.Columns) {
			return nil, fmt.Errorf("too many values for record element %s", val)
		}
		v := Value{t.Columns[i].Type, val}
		vals = append(vals, v)
	}
	return vals, nil
}

func (t *TypeRecord) Parse(in []byte) (zcode.Bytes, error) {
	panic("record.Parse shouldn't be called")
}

func (t *TypeRecord) StringOf(zv zcode.Bytes) string {
	d := "record["
	comma := ""
	it := zv.Iter()
	for _, col := range t.Columns {
		zv, _, err := it.Next()
		if err != nil {
			//XXX shouldn't happen
			d += "ERR"
			break
		}
		d += comma + Value{col.Type, zv}.String()
		comma = ","
	}
	d += "]"
	return d
}

func (t *TypeRecord) Marshal(zv zcode.Bytes) (interface{}, error) {
	m := make(map[string]Value)
	it := zv.Iter()
	for _, col := range t.Columns {
		zv, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		m[col.Name] = Value{col.Type, zv}
	}
	return m, nil
}
