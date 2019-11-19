package zeek

import (
	"encoding/json"
	"fmt"
)

type TypeOfEnum struct{}

func (t *TypeOfEnum) String() string {
	return "enum"
}

func (t *TypeOfEnum) Parse(value []byte) (string, error) {
	return string(value), nil
}

func (t *TypeOfEnum) Format(value []byte) (interface{}, error) {
	return string(value), nil
}

func (t *TypeOfEnum) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	return &Enum{Native: string(value)}, nil
}

type Enum struct {
	Native string
}

func (e *Enum) String() string {
	return e.Native
}

func (e *Enum) Type() Type {
	return TypeEnum
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a string or enum with the value's string value using a comparison
// based on op.
func (e *Enum) Comparison(op string) (Predicate, error) {
	compare, ok := compareString[op]
	if !ok {
		return nil, fmt.Errorf("unknown enum comparator: %s", op)
	}
	pattern := e.Native
	return func(typ Type, val []byte) bool {
		switch typ.(type) {
		case *TypeOfString, *TypeOfEnum:
			return compare(ustring(val), pattern)
		}
		return false
	}, nil
}

func (e *Enum) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfString:
		return &String{e.Native}
	case *TypeOfEnum:
		return e
	}
	return nil
}

func (e *Enum) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Native)
}

func (e *Enum) Elements() ([]Value, bool) { return nil, false }
