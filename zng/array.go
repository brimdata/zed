package zng

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
)

type TypeArray struct {
	id   int
	Type Type
}

func NewTypeArray(id int, typ Type) *TypeArray {
	return &TypeArray{id, typ}
}

func (t *TypeArray) ID() int {
	return t.id
}

//XXX get rid of this when we implement full ZNG
func (t *TypeArray) SetID(id int) {
	t.id = id
}

func (t *TypeArray) String() string {
	return fmt.Sprintf("array[%s]", t.Type)
}

func (t *TypeArray) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.Type, zv)
}

func (t *TypeArray) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeArray.Parse shouldn't be called")
}

func (t *TypeArray) StringOf(zv zcode.Bytes) string {
	s := "array["
	comma := ""
	it := zv.Iter()
	for !it.Done() {
		zv, container, err := it.Next()
		if container || err != nil {
			//XXX
			s += "ERR"
			break
		}
		s += comma + Value{t.Type, zv}.String()
		comma = ","
	}
	s += "]"
	return s
}

func (t *TypeArray) Marshal(zv zcode.Bytes) (interface{}, error) {
	// start out with zero-length container so we get "[]" instead of nil
	vals := make([]Value, 0)
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		vals = append(vals, Value{t.Type, val})
	}
	return vals, nil
}
