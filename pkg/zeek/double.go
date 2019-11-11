package zeek

import (
	"fmt"
	"strconv"

	"github.com/mccanne/zq/pkg/nano"
)

type TypeOfDouble struct{}

var compareFloat = map[string]func(float64, float64) bool{
	"eql":  func(a, b float64) bool { return a == b },
	"neql": func(a, b float64) bool { return a != b },
	"gt":   func(a, b float64) bool { return a > b },
	"gte":  func(a, b float64) bool { return a >= b },
	"lt":   func(a, b float64) bool { return a < b },
	"lte":  func(a, b float64) bool { return a <= b }}

func (s *TypeOfDouble) String() string {
	return "double"
}

func (t *TypeOfDouble) Parse(value []byte) (float64, error) {
	return UnsafeParseFloat64(value)
}

func (t *TypeOfDouble) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfDouble) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Double{Native: v}, nil
}

type Double struct {
	Native float64
}

func (d *Double) String() string {
	return strconv.FormatFloat(d.Native, 'g', -1, 64)
}

func (d *Double) Type() Type {
	return TypeDouble
}

// Comparison returns a Predicate that compares typed byte slices that must
// be coercible to an double with the value's double value using a comparison
// based on op.  Int, count, port, and double types can
// all be converted to the integer value.  XXX there are some overflow issues here.
func (d *Double) Comparison(op string) (Predicate, error) {
	compare, ok := compareFloat[op]
	if !ok {
		return nil, fmt.Errorf("unknown double comparator: %s", op)
	}
	pattern := d.Native
	return func(typ Type, val []byte) bool {
		switch typ := typ.(type) {
		// We allow comparison of float constant with integer-y
		// fields and just use typeDouble to parse since it will do
		// the right thing for integers.  XXX do we want to allow
		// integers that cause float64 overflow?  user can always
		// use an integer constant instead of a float constant to
		// compare with the integer-y field.
		case *TypeOfDouble, *TypeOfInt, *TypeOfCount, *TypeOfPort:
			v, err := TypeDouble.Parse(val)
			if err == nil {
				return compare(v, pattern)
			}
		case *TypeOfTime:
			ts, err := typ.Parse(val)
			if err == nil {
				return compare(float64(ts), pattern)
			}
		case *TypeOfInterval:
			v, err := typ.Parse(val)
			if err == nil {
				return compare(float64(v), pattern)
			}
		}
		return false
	}, nil
}

func (d *Double) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfDouble:
		return d
	case *TypeOfInt:
		return CoerceToInt(d)
	case *TypeOfCount:
		i := CoerceToInt(d)
		if i != nil && i.Native >= 0 {
			return &Count{uint64(i.Native)}
		}
	case *TypeOfPort:
		i := CoerceToInt(d)
		if i != nil && i.Native >= 0 && i.Native < 65536 {
			return &Port{uint32(i.Native)}
		}
	case *TypeOfTime:
		return &Time{nano.Ts(d.Native * 1e9)}
	case *TypeOfInterval:
		return &Interval{int64(d.Native * 1e9)}
	}
	return nil
}

// CoerceToDouble attempts to convert a value to a double and returns a new
// double value.  Time and Intervals are converted to an Int as
// their nanosecond values.  If input interface is already a Double, then
// it is returned as a *Double.  If the value cannot be coerced, then
// nil is returned.
// XXX this should be
func CoerceToDouble(in Value) *Double {
	switch v := in.(type) {
	case *Double:
		return v
	case *Int:
		return &Double{float64(v.Native)}
	case *Bool:
		if v.Native {
			return &Double{1}
		}
		return &Double{0}
	case *Count:
		return &Double{float64(v.Native)}
	case *Port:
		return &Double{float64(v.Native)}
	case *Time:
		return &Double{float64(v.Native)}
	case *Interval:
		return &Double{float64(v.Native)}
	}
	return nil
}

func (d *Double) Elements() ([]Value, bool) { return nil, false }
