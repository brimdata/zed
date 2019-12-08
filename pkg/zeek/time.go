package zeek

import (
	"encoding/json"
	"errors"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfTime struct{}

func (t *TypeOfTime) String() string {
	return "time"
}

func (t *TypeOfTime) Parse(value []byte) (nano.Ts, error) {
	if value == nil {
		return 0, ErrUnset
	}
	return nano.Parse(value)
}

func (t *TypeOfTime) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfTime) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := nano.Parse(value)
	if err != nil {
		return nil, err
	}
	return NewTime(v), nil
}

type Time nano.Ts

func NewTime(ts nano.Ts) *Time {
	t := Time(ts)
	return &t
}

func (t Time) String() string {
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano.Ts.  Such values cannot be representd by
	// float64's without loss of the least significant digits of ns,
	return nano.Ts(t).StringFloat()
}

func (t Time) Encode(dst zval.Encoding) zval.Encoding {
	v := []byte(t.String())
	return zval.AppendValue(dst, v)
}

func (t Time) Type() Type {
	return TypeTime
}

func (t Time) Comparison(op string) (Predicate, error) {
	// XXX we need to add time literals to zql before this matters
	return nil, errors.New("time comparisons not yet implemented")
}

func (t *Time) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfTime)
	if ok {
		return t
	}
	return nil
}

// CoerceToTime attempts to convert a value to a time and
// returns a new time value if the conversion is possible.  Int,
// is converted as nanoseconds and Double is converted as seconds. If
// the value cannot be coerced, then nil is returned.
func CoerceToTime(in Value, out *Time) bool {
	switch v := in.(type) {
	case *Time:
		*out = *v
		return true
	case *Int:
		*out = Time(*v)
		return true
	case *Double:
		s := *v * 1000 * 1000 * 1000
		*out = Time(s)
		return true
	}
	return false
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal((*nano.Ts)(t))
}

func (t *Time) Elements() ([]Value, bool) { return nil, false }
