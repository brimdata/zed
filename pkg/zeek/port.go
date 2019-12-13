package zeek

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfPort struct{}

func (t *TypeOfPort) String() string {
	return "port"
}

func EncodePort(p uint32) zval.Encoding {
	var b [2]byte
	b[0] = byte(p >> 8)
	b[1] = byte(p)
	return b[:]
}

func DecodePort(zv zval.Encoding) (uint32, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	if len(zv) != 2 {
		return 0, errors.New("port encoding must be 2 bytes")

	}
	return uint32(zv[0])<<8 | uint32(zv[1]), nil
}

func (t *TypeOfPort) Parse(in []byte) (zval.Encoding, error) {
	i, err := UnsafeParseUint32(in)
	if err != nil {
		return nil, err
	}
	return EncodePort(i), nil
}

func (t *TypeOfPort) New(zv zval.Encoding) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	v, err := DecodePort(zv)
	if err != nil {
		return nil, err
	}
	return NewPort(v), nil
}

type Port uint32

func NewPort(p uint32) *Port {
	v := Port(p)
	return &v
}

func (p Port) String() string {
	return strconv.FormatUint(uint64(p), 10)
}

func (p Port) Type() Type {
	return TypePort
}

func (p Port) Encode(dst zval.Encoding) zval.Encoding {
	return zval.AppendValue(dst, EncodePort(uint32(p)))

}

// Comparison returns a Predicate that compares typed byte slices that must
// be a port with the value's port value using a comparison based on op.
// Integer fields are not coerced (nor are any other types) so they never
// match the port literal here.
func (p Port) Comparison(op string) (Predicate, error) {
	compare, ok := compareInt[op]
	if !ok {
		return nil, fmt.Errorf("unknown port comparator: %s", op)
	}
	// only a zeek port can be compared with a port type.  If the user went
	// to the trouble of specifying a port match (e.g., ":80" vs "80") then
	// we use strict typing here on the port comparison.
	pattern := int64(p)
	return func(e TypedEncoding) bool {
		if _, ok := e.Type.(*TypeOfPort); !ok {
			return false
		}
		v, err := DecodePort(e.Body)
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
	return json.Marshal((*uint32)(p))
}

func (p Port) Elements() ([]Value, bool) { return nil, false }
