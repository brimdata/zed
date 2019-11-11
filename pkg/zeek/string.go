package zeek

import "fmt"

type TypeOfString struct{}

var compareString = map[string]func(string, string) bool{
	"eql":  func(a, b string) bool { return a == b },
	"neql": func(a, b string) bool { return a != b },
	"gt":   func(a, b string) bool { return a > b },
	"gte":  func(a, b string) bool { return a >= b },
	"lt":   func(a, b string) bool { return a < b },
	"lte":  func(a, b string) bool { return a <= b }}

func (t *TypeOfString) String() string {
	return "string"
}

func (t *TypeOfString) Parse(value []byte) (string, error) {
	return string(value), nil
}

func (t *TypeOfString) Format(value []byte) (interface{}, error) {
	return string(value), nil
}

func (t *TypeOfString) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	return &String{Native: string(value)}, nil
}

type String struct {
	Native string
}

func (s *String) String() string {
	return s.Native
}

func (s *String) Type() Type {
	return TypeString
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a string or enum with the value's string value using a comparison
// based on op.
func (s *String) Comparison(op string) (Predicate, error) {
	compare, ok := compareString[op]
	if !ok {
		return nil, fmt.Errorf("unknown string comparator: %s", op)
	}
	pattern := s.Native
	return func(typ Type, val []byte) bool {
		switch typ.(type) {
		case *TypeOfString, *TypeOfEnum:
			return compare(ustring(val), pattern)
		}
		return false
	}, nil
}

func (s *String) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfString:
		return s
	case *TypeOfEnum:
		return &Enum{s.Native}
	}
	return nil
}

func (s *String) Elements() ([]Value, bool) { return nil, false }
