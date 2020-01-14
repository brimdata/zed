package zng

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
)

type TypeVector struct {
	typ Type
}

func (t *TypeVector) String() string {
	return fmt.Sprintf("vector[%s]", t.typ)
}

// parse a vector body type of the form "[type]"
func parseVectorTypeBody(in string) (string, *TypeVector, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, ErrTypeSyntax
	}
	var typ Type
	var err error
	rest, typ, err = parseType(rest)
	if err != nil {
		return "", nil, err
	}
	rest, ok = match(rest, "]")
	if !ok {
		return "", nil, ErrTypeSyntax
	}
	return rest, &TypeVector{typ: typ}, nil
}

func (t *TypeVector) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.typ, zv)
}

func (t *TypeVector) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeVector.Parse shouldn't be called")
}

func (t *TypeVector) StringOf(zv zcode.Bytes) string {
	s := "vector["
	comma := ""
	it := zv.Iter()
	for !it.Done() {
		zv, container, err := it.Next()
		if container || err != nil {
			//XXX
			s += "ERR"
			break
		}
		s += comma + Value{t.typ, zv}.String()
		comma = ","
	}
	s += "]"
	return s
}

func (t *TypeVector) Marshal(zv zcode.Bytes) (interface{}, error) {
	// start out with zero-length container so we get "[]" instead of nil
	vals := make([]Value, 0)
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		vals = append(vals, Value{t.typ, val})
	}
	return vals, nil
}
