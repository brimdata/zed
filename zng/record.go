package zng

import (
	"encoding/json"
	"fmt"

	"github.com/mccanne/zq/zcode"
)

type TypeRecord struct {
	Context
	ID      int
	Columns []Column
	LUT     map[string]int
	TsCol   int
	//XXX get rid of Key and use ID as context-unique type id
	Key string
}

func CopyTypeRecord(id int, r *TypeRecord) *TypeRecord {
	return &TypeRecord{
		Columns: r.Columns,
		LUT:     r.LUT,
		TsCol:   r.TsCol,
		Key:     r.Key,
	}
}

func NewTypeRecord(id int, columns []Column) *TypeRecord {
	r := &TypeRecord{
		ID:      id,
		Columns: columns,
		TsCol:   -1,
		Key:     ColumnString("", columns, ""), //XXX
	}
	r.createLUT()
	return r
}

//XXX
func TypeRecordString(columns []Column) string {
	return ColumnString("record[", columns, "]")
}

func (t *TypeRecord) String() string {
	return ColumnString("record[", t.Columns, "]")
}

func (t TypeRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Columns)
}

func (t *TypeRecord) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &t.Columns); err != nil {
		return err
	}
	Typify(t.Context, t.Columns)
	return nil
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

func (t *TypeRecord) ColumnOfField(field string) (int, bool) {
	v, ok := t.LUT[field]
	return v, ok
}

func (t *TypeRecord) HasField(field string) bool {
	_, ok := t.LUT[field]
	return ok
}

func (t *TypeRecord) createLUT() {
	t.LUT = make(map[string]int)
	for k, col := range t.Columns {
		t.LUT[col.Name] = k
		if col.Name == "ts" {
			if _, ok := col.Type.(*TypeOfTime); ok {
				t.TsCol = k
			}
		}
	}
}
