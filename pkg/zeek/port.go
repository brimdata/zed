package zeek

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type TypeOfPort struct{}

func (t *TypeOfPort) String() string {
	return "port"
}

func (t *TypeOfPort) Parse(value []byte) (uint32, error) {
	if value == nil {
		return 0, ErrUnset
	}
	return UnsafeParseUint32(value)
}

func (t *TypeOfPort) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfPort) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Port{Native: uint32(v)}, nil
}

type Port struct {
	Native uint32
}

func (p *Port) String() string {
	return strconv.FormatUint(uint64(p.Native), 10)
}

func (p *Port) Type() Type {
	return TypePort
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a port with the value's port value using a comparison based on op.
// Integer fields are not coerced (nor are any other types) so they never
// match the port literal here.
func (p *Port) Comparison(op string) (Predicate, error) {
	compare, ok := compareInt[op]
	if !ok {
		return nil, fmt.Errorf("unknown port comparator: %s", op)
	}
	// only a zeek port can be compared with a port type.  If the user went
	// to the trouble of specifying a port match (e.g., ":80" vs "80") then
	// we use strict typing here on the port comparison.
	pattern := int64(p.Native)
	return func(typ Type, val []byte) bool {
		typePort, ok := typ.(*TypeOfPort)
		if !ok {
			return false
		}
		v, err := typePort.Parse(val)
		if err != nil {
			return false
		}
		return compare(int64(v), pattern)
	}, nil
}

func (p *Port) Coerce(typ Type) Value {
	// ints can be turned into ports but ports can't be turned into ints
	_, ok := typ.(*TypeOfPort)
	if ok {
		return p
	}
	return nil
}

func (p *Port) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Native)
}

func (p *Port) Elements() ([]Value, bool) { return nil, false }
