package zng

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/zcode"
)

type TypeOfString struct{}

var compareString = map[string]func(string, string) bool{
	"eql":    func(a, b string) bool { return a == b },
	"neql":   func(a, b string) bool { return a != b },
	"gt":     func(a, b string) bool { return a > b },
	"gte":    func(a, b string) bool { return a >= b },
	"lt":     func(a, b string) bool { return a < b },
	"lte":    func(a, b string) bool { return a <= b },
	"search": func(a, b string) bool { return strings.Contains(a, b) },
}

func (t *TypeOfString) String() string {
	return "string"
}

func EncodeString(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeString(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfString) Parse(in []byte) (zcode.Bytes, error) {
	return in, nil
}

func (t *TypeOfString) New(zv zcode.Bytes) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	return NewString(string(zv)), nil
}

type String string

func NewString(s string) *String {
	v := String(s)
	return &v
}

func (s String) String() string {
	return string(s)
}

func (s String) Encode(dst zcode.Bytes) zcode.Bytes {
	v := []byte(s)
	return zcode.AppendValue(dst, v)
}

func (s String) Type() Type {
	return TypeString
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a string or enum with the value's string value using a comparison
// based on op.
func (s String) Comparison(op string) (Predicate, error) {
	compare, ok := compareString[op]
	if !ok {
		return nil, fmt.Errorf("unknown string comparator: %s", op)
	}
	pattern := string(s)
	return func(e TypedEncoding) bool {
		switch e.Type.(type) {
		case *TypeOfString, *TypeOfEnum:
			return compare(ustring(e.Body), pattern)
		}
		return false
	}, nil
}

func (s *String) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfString:
		return s
	case *TypeOfEnum:
		return NewEnum(string(*s))
	}
	return nil
}

func (s String) Elements() ([]Value, bool) { return nil, false }
