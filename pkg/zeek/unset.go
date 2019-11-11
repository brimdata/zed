package zeek

import "fmt"

type TypeOfUnset struct{}

var compareUnset = map[string]func([]byte) bool{
	"eql":  func(val []byte) bool { return val == nil },
	"neql": func(val []byte) bool { return val != nil },
}

func (n *TypeOfUnset) String() string {
	return "none"
}

func (t *TypeOfUnset) Parse(value []byte) (string, error) {
	return "none", nil
}

func (t *TypeOfUnset) Format(value []byte) (interface{}, error) {
	return "none", nil
}
func (n *TypeOfUnset) New(value []byte) (Value, error) {
	return &Unset{}, nil
}

type Unset struct{}

func (n *Unset) String() string {
	return "-"
}

func (n *Unset) Type() Type {
	return TypeUnset
}

func (n *Unset) Comparison(op string) (Predicate, error) {
	compare, ok := compareUnset[op]
	if !ok {
		return nil, fmt.Errorf("unknown unset comparator: %s", op)
	}
	return func(typ Type, val []byte) bool {
		switch typ.(type) {
		case *TypeOfString, *TypeOfBool, *TypeOfCount, *TypeOfInt, *TypeOfDouble, *TypeOfTime, *TypeOfInterval, *TypeOfPort, *TypeOfAddr, *TypeOfSubnet, *TypeOfEnum, *TypeSet, *TypeVector:
			return compare(val)
		}
		return false
	}, nil
}

func (n *Unset) Coerce(typ Type) Value {
	return nil
}

func (n *Unset) Elements() ([]Value, bool) { return nil, false }
