package zng

import (
	"encoding/json"
	"errors"
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
	return rest, &TypeVector{typ}, nil
}

type Vector struct {
	typ    *TypeVector
	values []Value
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

func (t *TypeVector) New(zv zcode.Bytes) (Value, error) {
	if zv == nil {
		return &Vector{typ: t, values: nil}, nil
	}
	v, err := t.Decode(zv)
	if err != nil {
		return nil, err
	}
	return &Vector{typ: t, values: v}, nil
}

func (v *Vector) String() string {
	s := "vector["
	comma := ""
	for _, item := range v.values {
		s += comma + item.String()
		comma = ","
	}
	s += "]"
	return s
}

func (v *Vector) Encode(dst zcode.Bytes) zcode.Bytes {
	zv := make(zcode.Bytes, 0)
	for _, val := range v.values {
		zv = val.Encode(zv)
	}
	return zcode.AppendContainer(dst, zv)
}

func (v *Vector) Type() Type {
	return v.typ
}

func (v *Vector) Comparison(op string) (Predicate, error) {
	return nil, errors.New("no support yet for vector comparison")
}

func (v *Vector) Coerce(typ Type) Value {
	_, ok := typ.(*TypeVector)
	if ok {
		return v
	}
	return nil
}

func (v *Vector) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.values)
}

func (v *Vector) Elements() ([]Value, bool) { return v.values, true }
