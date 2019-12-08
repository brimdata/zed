package zeek

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfDouble struct{}

var compareFloat = map[string]func(float64, float64) bool{
	"eql":  func(a, b float64) bool { return a == b },
	"neql": func(a, b float64) bool { return a != b },
	"gt":   func(a, b float64) bool { return a > b },
	"gte":  func(a, b float64) bool { return a >= b },
	"lt":   func(a, b float64) bool { return a < b },
	"lte":  func(a, b float64) bool { return a <= b }}

func (t *TypeOfDouble) String() string {
	return "double"
}

func (t *TypeOfDouble) Parse(value []byte) (float64, error) {
	if value == nil {
		return 0, ErrUnset
	}
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
	return NewDouble(v), nil
}

type Double float64

func NewDouble(f float64) *Double {
	d := Double(f)
	return &d
}

func (d Double) String() string {
	return strconv.FormatFloat(float64(d), 'g', -1, 64)
}

func (d *Double) Encode(dst zval.Encoding) zval.Encoding {
	v := []byte(d.String())
	return zval.AppendValue(dst, v)
}

func (d *Double) Type() Type {
	return TypeDouble
}

// Comparison returns a Predicate that compares typed byte slices that must
// be coercible to an double with the value's double value using a comparison
// based on op.  Int, count, port, and double types can
// all be converted to the integer value.  XXX there are some overflow issues here.
func (d Double) Comparison(op string) (Predicate, error) {
	compare, ok := compareFloat[op]
	if !ok {
		return nil, fmt.Errorf("unknown double comparator: %s", op)
	}
	pattern := float64(d)
	return func(e TypedEncoding) bool {
		val := e.Body
		switch typ := e.Type.(type) {
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
		var i Int
		if CoerceToInt(d, &i) {
			return &i
		}
		return nil
	case *TypeOfCount:
		var i Int
		if CoerceToInt(d, &i) && i >= 0 {
			return NewCount(uint64(i))
		}
		return nil

	case *TypeOfPort:
		var i Int
		if CoerceToInt(d, &i) && i >= 0 && i < 65536 {
			return NewPort(uint32(i))
		}
	case *TypeOfTime:
		return NewTime(nano.Ts(*d * 1e9))
	case *TypeOfInterval:
		return NewInterval(int64(*d * 1e9))
	}
	return nil
}

// CoerceToDouble attempts to convert a value to a double and returns a new
// double value.  Time and Intervals are converted to an Int as
// their nanosecond values.  If input interface is already a Double, then
// it is returned as a *Double.  If the value cannot be coerced, then
// nil is returned.
func CoerceToDouble(in Value) *Double {
	switch v := in.(type) {
	case *Double:
		return v
	case *Int:
		return NewDouble(float64(*v))
	case *Bool:
		if *v {
			return NewDouble(1)
		}
		return NewDouble(0)
	case *Count:
		return NewDouble(float64(*v))
	case *Port:
		return NewDouble(float64(*v))
	case *Time:
		return NewDouble(float64(*v))
	case *Interval:
		return NewDouble(float64(*v))
	}
	return nil
}

func (d *Double) MarshalJSON() ([]byte, error) {
	return json.Marshal((*float64)(d))
}

func (d Double) Elements() ([]Value, bool) { return nil, false }
