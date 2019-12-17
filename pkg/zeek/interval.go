package zeek

import (
	"errors"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfInterval struct{}

func (t *TypeOfInterval) String() string {
	return "interval"
}

func (t *TypeOfInterval) Parse(in []byte) (zval.Encoding, error) {
	dur, err := nano.ParseDuration(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(int64(dur)), nil
}

func (t *TypeOfInterval) New(zv zval.Encoding) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	v, err := DecodeInt(zv)
	if err != nil {
		return nil, err
	}
	return NewInterval(v), nil
}

type Interval int64

func NewInterval(i int64) *Interval {
	p := Interval(i)
	return &p
}

func (i Interval) String() string {
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano Duration. Such values cannot be represented by
	// float64's without loss of the least significant digits of ns,
	return nano.DurationString(int64(i))
}

func (i Interval) Encode(dst zval.Encoding) zval.Encoding {
	return zval.AppendValue(dst, EncodeInt(int64(i)))
}

func DecodeInterval(zv zval.Encoding) (int64, error) {
	return DecodeInt(zv)
}

func (i Interval) Type() Type {
	return TypeInterval
}

func (i Interval) Comparison(op string) (Predicate, error) {
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
func CoerceToInterval(in Value, out *Interval) bool {
	switch v := in.(type) {
	case *Interval:
		*out = *v
		return true
	case *Int:
		*out = Interval(int64(*v))
		return true
	case *Double:
		s := *v * 1000_000_000
		*out = Interval(int64(s))
		return true
	}
	return false
}

func (i *Interval) Elements() ([]Value, bool) { return nil, false }
