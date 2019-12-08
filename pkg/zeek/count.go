package zeek

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfCount struct{}

func (t *TypeOfCount) String() string {
	return "count"
}

func (t *TypeOfCount) Parse(value []byte) (uint64, error) {
	if value == nil {
		return 0, ErrUnset
	}
	return UnsafeParseUint64(value)
}

func (t *TypeOfCount) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfCount) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return NewCount(v), nil
}

type Count uint64

func NewCount(c uint64) *Count {
	p := Count(c)
	return &p
}

func (c Count) String() string {
	return strconv.FormatUint(uint64(c), 10)
}

func (c Count) Encode(dst zval.Encoding) zval.Encoding {
	v := []byte(c.String())
	return zval.AppendValue(dst, v)
}

func (c Count) Type() Type {
	return TypeCount
}

// Comparison returns an error since count literals currently aren't supported.
// If we add a count literal syntax to the language at some point, we can
// fix this.
func (c *Count) Comparison(op string) (Predicate, error) {
	return nil, errors.New("literal count types are not supported")
}

func (c *Count) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfCount)
	if ok {
		return c
	}
	return nil
}

func (c *Count) MarshalJSON() ([]byte, error) {
	return json.Marshal((*uint64)(c))
}

func (c *Count) Elements() ([]Value, bool) { return nil, false }
