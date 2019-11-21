package zeek

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type TypeOfBool struct{}

var compareBool = map[string]func(bool, bool) bool{
	"eql":  func(a, b bool) bool { return a == b },
	"neql": func(a, b bool) bool { return a != b },
	"gt": func(a, b bool) bool {
		if a {
			return !b
		}
		return false
	},
	"gte": func(a, b bool) bool {
		if a {
			return true
		}
		return !b
	},
	"lt": func(a, b bool) bool {
		if a {
			return false
		}
		return b
	},
	"lte": func(a, b bool) bool {
		if a {
			return b
		}
		return !b
	},
}

func (t *TypeOfBool) String() string {
	return "bool"
}

func (t *TypeOfBool) Parse(value []byte) (bool, error) {
	if value == nil {
		return false, ErrUnset
	}
	return UnsafeParseBool(value)
}

func (t *TypeOfBool) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfBool) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Bool{Native: v}, nil
}

type Bool struct {
	Native bool
}

func (b *Bool) String() string {
	return strconv.FormatBool(b.Native)
}

func (b *Bool) Type() Type {
	return TypeBool
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a boolean or coercible to an integer.  In the later case, the integer
// is converted to a boolean.
func (b *Bool) Comparison(op string) (Predicate, error) {
	compare, ok := compareBool[op]
	if !ok {
		return nil, fmt.Errorf("unknown bool comparator: %s", op)
	}
	pattern := b.Native
	return func(typ Type, val []byte) bool {
		typeBool, ok := typ.(*TypeOfBool)
		if !ok {
			return false
		}
		v, err := typeBool.Parse(val)
		if err != nil {
			return false
		}
		return compare(v, pattern)
	}, nil
	return nil, fmt.Errorf("bad comparator for boolean type: %s", op)
}

func (b *Bool) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfBool)
	if ok {
		return b
	}
	return nil
}

func (b *Bool) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.Native)
}

func (b *Bool) Elements() ([]Value, bool) { return nil, false }
