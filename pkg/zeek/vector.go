package zeek

import (
	"errors"
	"fmt"
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

func (t *TypeVector) Parse(b []byte) ([]Value, error) {
	return parseContainer(t, t.typ, b)
}

func (t *TypeVector) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeVector) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
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

func (v *Vector) Elements() ([]Value, bool) { return v.values, true }
