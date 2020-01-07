package zng

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
)

type TypeOfEnum struct{}

func (t *TypeOfEnum) String() string {
	return "enum"
}

func EncodeEnum(e []byte) zcode.Bytes {
	return e
}

func DecodeEnum(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfEnum) Parse(in []byte) (zcode.Bytes, error) {
	return in, nil
}

func (t *TypeOfEnum) New(zv zcode.Bytes) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	return NewEnum(string(zv)), nil
}

type Enum string

func NewEnum(s string) *Enum {
	p := Enum(s)
	return &p
}

func (e Enum) String() string {
	return string(e)
}

func (e Enum) Encode(dst zcode.Bytes) zcode.Bytes {
	return zcode.AppendValue(dst, EncodeEnum([]byte(e)))
}

func (e Enum) Type() Type {
	return TypeEnum
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a string or enum with the value's string value using a comparison
// based on op.
func (e Enum) Comparison(op string) (Predicate, error) {
	compare, ok := compareString[op]
	if !ok {
		return nil, fmt.Errorf("unknown enum comparator: %s", op)
	}
	pattern := string(e)
	return func(e TypedEncoding) bool {
		switch e.Type.(type) {
		case *TypeOfString, *TypeOfEnum:
			return compare(ustring(e.Body), pattern)
		}
		return false
	}, nil
}

func (e *Enum) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfString:
		return NewString(string(*e))
	case *TypeOfEnum:
		return e
	}
	return nil
}

func (e *Enum) Elements() ([]Value, bool) { return nil, false }
