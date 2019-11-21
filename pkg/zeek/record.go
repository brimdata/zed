package zeek

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mccanne/zq/pkg/zval"
)

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

func (t *TypeRecord) Parse(b []byte) ([]Value, error) {
	if b == nil {
		return nil, ErrUnset
	}
	var vals []Value
	for i, it := 0, zval.Iter(b); !it.Done(); i++ {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		if i >= len(t.Columns) {
			return nil, fmt.Errorf("too many values for record element %s", val)
		}
		v, err := t.Columns[i].Type.New(val)
		if err != nil {
			return nil, fmt.Errorf("cannot parse record element %s", val)
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func (t *TypeRecord) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeRecord) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Record{typ: t, values: v}, nil
}

type Record struct {
	typ    *TypeRecord
	values []Value
}

func (r *Record) String() string {
	//XXX this should just be the values no?  need to change set and vector too
	d := "record["
	comma := ""
	for _, item := range r.values {
		d += comma + item.String()
		comma = ","
	}
	d += "]"
	return d
}

func (r *Record) Type() Type {
	return r.typ
}

func (r *Record) Comparison(op string) (Predicate, error) {
	return nil, errors.New("no support yet for record comparison")
}

func (r *Record) Coerce(typ Type) Value {
	_, ok := typ.(*TypeRecord)
	if ok {
		return r
	}
	return nil
}

func (r *Record) MarshalJSON() ([]byte, error) {
	m := make(map[string]Value)
	for i, col := range r.typ.Columns {
		m[col.Name] = r.values[i]
	}
	return json.Marshal(m)
}

func (r *Record) Elements() ([]Value, bool) { return r.values, true }
