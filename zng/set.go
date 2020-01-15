package zng

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
)

type TypeSet struct {
	InnerType Type
}

func (t *TypeSet) String() string {
	return fmt.Sprintf("set[%s]", t.InnerType)
}

func (t *TypeSet) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.InnerType, zv)
}

func (t *TypeSet) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeSet.Parse shouldn't be called")
}

func (t *TypeSet) StringOf(zv zcode.Bytes) string {
	d := "set["
	comma := ""
	it := zv.Iter()
	for !it.Done() {
		val, container, err := it.Next()
		if container || err != nil {
			//XXX
			d += "ERR"
			break
		}
		d += comma + t.InnerType.StringOf(val)
		comma = ","
	}
	d += "]"
	return d
}

func (t *TypeSet) Marshal(zv zcode.Bytes) (interface{}, error) {
	// start out with zero-length container so we get "[]" instead of nil
	vals := make([]Value, 0)
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		vals = append(vals, Value{t.InnerType, val})
	}
	return vals, nil
}
