package zng

import (
	"errors"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
)

type TypeOfTime struct{}

func (t *TypeOfTime) String() string {
	return "time"
}

func EncodeTime(t nano.Ts) zcode.Bytes {
	var b [8]byte
	n := encodeInt(b[:], int64(t))
	return b[:n]
}

func DecodeTime(zv zcode.Bytes) (nano.Ts, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return nano.Ts(decodeInt(zv)), nil
}

func (t *TypeOfTime) Parse(in []byte) (zcode.Bytes, error) {
	ts, err := nano.Parse(in)
	if err != nil {
		return nil, err
	}
	return EncodeTime(ts), nil
}

func (t *TypeOfTime) New(value zcode.Bytes) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	ts, err := DecodeTime(value)
	if err != nil {
		return nil, err
	}
	return NewTime(ts), nil
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

func (t Time) Encode(dst zcode.Bytes) zcode.Bytes {
	return zcode.AppendSimple(dst, EncodeTime(nano.Ts(t)))
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

// CoerceToTime attempts to convert a value to a time. Int and Double
// are converted as seconds. The resulting coerced value is written to
// out, and true is returned. If the value cannot be coerced, then
// false is returned.
func CoerceToTime(in Value, out *Time) bool {
	switch v := in.(type) {
	case *Time:
		*out = *v
		return true
	case *Int:
		s := *v * 1e9
		*out = Time(s)
		return true
	case *Double:
		s := *v * 1e9
		*out = Time(s)
		return true
	}
	return false
}

func (t *Time) Elements() ([]Value, bool) { return nil, false }
