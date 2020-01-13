package zng

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
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

func EncodeBool(b bool) zcode.Bytes {
	var v [1]byte
	if b {
		v[0] = 1
	}
	return v[:]
}

func DecodeBool(zv zcode.Bytes) (bool, error) {
	if zv == nil {
		return false, ErrUnset
	}
	if zv[0] != 0 {
		return true, nil
	}
	return false, nil
}

func (t *TypeOfBool) Parse(in []byte) (zcode.Bytes, error) {
	b, err := UnsafeParseBool(in)
	if err != nil {
		return nil, err
	}
	return EncodeBool(b), nil
}

func (t *TypeOfBool) New(zv zcode.Bytes) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	v, err := DecodeBool(zv)
	if err != nil {
		return nil, err
	}
	return NewBool(v), nil
}

type Bool bool

func NewBool(b bool) *Bool {
	p := Bool(b)
	return &p
}

func (b Bool) String() string {
	if b {
		return "T"
	}
	return "F"
}

func (b Bool) Encode(dst zcode.Bytes) zcode.Bytes {
	return zcode.AppendSimple(dst, EncodeBool(bool(b)))
}

func (b Bool) Type() Type {
	return TypeBool
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a boolean or coercible to an integer.  In the later case, the integer
// is converted to a boolean.
func (b Bool) Comparison(op string) (Predicate, error) {
	compare, ok := compareBool[op]
	if !ok {
		return nil, fmt.Errorf("unknown bool comparator: %s", op)
	}
	pattern := bool(b)
	return func(e TypedEncoding) bool {
		if _, ok := e.Type.(*TypeOfBool); !ok {
			return false
		}
		v, err := DecodeBool(e.Body)
		if err != nil {
			return false
		}
		return compare(v, pattern)
	}, nil
	return nil, fmt.Errorf("bad comparator for boolean type: %s", op)
}

func (b Bool) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfBool)
	if ok {
		return b
	}
	return nil
}

func (b Bool) Elements() ([]Value, bool) { return nil, false }
