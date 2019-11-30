package zeek

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mccanne/zq/pkg/zval"
)

type TypeSet struct {
	innerType Type
}

func (t *TypeSet) String() string {
	return fmt.Sprintf("set[%s]", t.innerType)
}

// parseSetTypeBody parses a set type body of the form "[type]" presuming the set
// keyword is already matched.
// The syntax "set[type1,type2,...]" for set-of-vectors is not supported.
func parseSetTypeBody(in string) (string, Type, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, ErrTypeSyntax
	}
	in = rest
	var types []Type
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, typ, err := parseType(in)
		if err != nil {
			return "", nil, err
		}
		types = append(types, typ)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if !ok {
			return "", nil, ErrTypeSyntax
		}
		if len(types) > 1 {
			return "", nil, fmt.Errorf("sets with multiple type parameters")
		}
		return rest, &TypeSet{types[0]}, nil
	}
}

func (t *TypeSet) Parse(b []byte) ([]Value, error) {
	if b == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.innerType, b)
}

func (t *TypeSet) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeSet) New(value []byte) (Value, error) {
	if value == nil {
		return &Set{typ: t, values: []Value{}}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Set{typ: t, values: v}, nil
}

type Set struct {
	typ    *TypeSet
	values []Value
}

func (s *Set) String() string {
	d := "set["
	comma := ""
	for _, item := range s.values {
		d += comma + item.String()
		comma = ","
	}
	d += "]"
	return d
}

func (s *Set) Encode(dst zval.Encoding) zval.Encoding {
	zv := make(zval.Encoding, 0)
	for _, v := range s.values {
		zv = v.Encode(zv)
	}
	return zval.AppendContainerValue(dst, zv)
}

func (s *Set) Type() Type {
	return s.typ
}

func (s *Set) Comparison(op string) (Predicate, error) {
	return nil, errors.New("no support yet for set comparison")
}

func (s *Set) Coerce(typ Type) Value {
	_, ok := typ.(*TypeSet)
	if ok {
		return s
	}
	return nil
}

func (s *Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.values)
}

func (s *Set) Elements() ([]Value, bool) { return s.values, true }
