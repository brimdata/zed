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

func EncodeCount(c uint64) zval.Encoding {
	var b [8]byte
	n := encodeUint(b[:], uint64(c))
	return b[:n]
}

func DecodeCount(zv zval.Encoding) (Count, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return Count(decodeUint(zv)), nil
}

func (t *TypeOfCount) Parse(in []byte) (zval.Encoding, error) {
	c, err := UnsafeParseUint64(in)
	if err != nil {
		return nil, err
	}
	return EncodeCount(c), nil
}

func (t *TypeOfCount) New(zv zval.Encoding) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	v, err := DecodeCount(zv)
	if err != nil {
		return nil, err
	}
	return NewCount(uint64(v)), nil
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
	return zval.AppendValue(dst, EncodeCount(uint64(c)))
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
