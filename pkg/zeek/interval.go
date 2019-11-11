package zeek

import (
	"errors"

	"github.com/mccanne/zq/pkg/nano"
)

type TypeOfInterval struct{}

func (t *TypeOfInterval) String() string {
	return "interval"
}

func (t *TypeOfInterval) Parse(value []byte) (int64, error) {
	return nano.ParseDuration(value)
}

func (t *TypeOfInterval) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfInterval) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := nano.ParseDuration(value)
	if err != nil {
		return nil, err
	}
	return &Interval{Native: v}, nil
}

type Interval struct {
	Native int64
}

func (i *Interval) String() string {
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano Duration. Such values cannot be represented by
	// float64's without loss of the least significant digits of ns,
	return nano.DurationString(i.Native)
}

func (i *Interval) Type() Type {
	return TypeInterval
}

func (i *Interval) Comparison(op string) (Predicate, error) {
	// XXX we need to add time/interval literals to zql before this matters
	return nil, errors.New("interval comparisons not yet implemented")
}

func (i *Interval) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfInterval)
	if ok {
		return i
	}
	return nil
}

// CoerceToInterval attempts to convert a value to an interval and
// returns a new interval value if the conversion is possible.  Int,
// is converted as nanoseconds and Double is converted as seconds. If
// the value cannot be coerced, then nil is returned.
func CoerceToInterval(in Value) *Interval {
	switch v := in.(type) {
	case *Interval:
		return v
	case *Int:
		return &Interval{v.Native}
	case *Double:
		s := v.Native * 1000 * 1000 * 1000
		return &Interval{int64(s)}
	}
	return nil
}

func (i *Interval) Elements() ([]Value, bool) { return nil, false }
