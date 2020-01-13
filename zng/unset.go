package zng

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mccanne/zq/zcode"
)

// ErrUnset is returned in Type.Parse / Type.Format when the value is unset.
var ErrUnset = errors.New("value is unset")

type TypeOfUnset struct{}

var compareUnset = map[string]func([]byte) bool{
	"eql":  func(val []byte) bool { return val == nil },
	"neql": func(val []byte) bool { return val != nil },
}

func (t *TypeOfUnset) String() string {
	return "none"
}

func (t *TypeOfUnset) Parse(in []byte) (zcode.Bytes, error) {
	return nil, nil
}

func (t *TypeOfUnset) Format(value []byte) (interface{}, error) {
	return "none", nil
}
func (t *TypeOfUnset) New(value zcode.Bytes) (Value, error) {
	return &Unset{}, nil
}

type Unset struct{}

func (u Unset) String() string {
	return "-"
}

func (u Unset) Encode(dst zcode.Bytes) zcode.Bytes {
	return zcode.AppendSimple(dst, nil)
}

func (u Unset) Type() Type {
	return TypeUnset
}

func (u Unset) Comparison(op string) (Predicate, error) {
	compare, ok := compareUnset[op]
	if !ok {
		return nil, fmt.Errorf("unknown unset comparator: %s", op)
	}
	return func(e TypedEncoding) bool {
		switch e.Type.(type) {
		case *TypeOfString, *TypeOfBool, *TypeOfCount, *TypeOfInt, *TypeOfDouble, *TypeOfTime, *TypeOfInterval, *TypeOfPort, *TypeOfAddr, *TypeOfSubnet, *TypeOfEnum, *TypeSet, *TypeVector:
			return compare(e.Body)
		}
		return false
	}, nil
}

func (u *Unset) Coerce(typ Type) Value {
	return nil
}

func (u *Unset) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}

func (u Unset) Elements() ([]Value, bool) { return nil, false }
