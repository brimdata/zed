package zeek

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mccanne/zq/pkg/nano"
)

type TypeOfInt struct{}

var compareInt = map[string]func(int64, int64) bool{
	"eql":  func(a, b int64) bool { return a == b },
	"neql": func(a, b int64) bool { return a != b },
	"gt":   func(a, b int64) bool { return a > b },
	"gte":  func(a, b int64) bool { return a >= b },
	"lt":   func(a, b int64) bool { return a < b },
	"lte":  func(a, b int64) bool { return a <= b }}

func (i *TypeOfInt) String() string {
	return "int"
}

func (t *TypeOfInt) Parse(value []byte) (int64, error) {
	return UnsafeParseInt64(value)
}

func (t *TypeOfInt) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfInt) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	v, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Int{Native: v}, nil
}

type Int struct {
	Native int64
}

func (i *Int) String() string {
	return strconv.FormatInt(i.Native, 10)
}

func (i *Int) Type() Type {
	return TypeInt
}

func (i *Int) NativeComparison(op string) (func(int64) bool, error) {
	compare, ok := compareInt[op]
	if !ok {
		return nil, fmt.Errorf("unknown comparator: %s", op)
	}
	return func(val int64) bool {
		return compare(val, i.Native)
	}, nil
}

// Comparison returns a Predicate that compares typed byte slices that must
// be coercible to an integer with the value's integer value using a comparison
// based on op.  Int, count, port, double, bool, time and interval types can
// all be converted to the integer value.  XXX there are some overflow issues here.
func (i *Int) Comparison(op string) (Predicate, error) {
	CompareInt, ok1 := compareInt[op]
	CompareFloat, ok2 := compareFloat[op]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown int comparator: %s", op)
	}
	pattern := i.Native
	// many different zeek data types can be compared with integers
	return func(typ Type, val []byte) bool {
		switch typ := typ.(type) {
		case *TypeOfInt, *TypeOfCount, *TypeOfPort:
			// we can parse counts and ports as an integer
			v, err := TypeInt.Parse(val)
			if err == nil {
				return CompareInt(v, pattern)
			}
		case *TypeOfDouble:
			v, err := typ.Parse(val)
			if err == nil {
				return CompareFloat(v, float64(pattern))
			}
		case *TypeOfTime:
			ts, err := typ.Parse(val)
			if err == nil {
				return CompareInt(int64(ts), pattern)
			}
		case *TypeOfInterval:
			v, err := typ.Parse(val)
			if err == nil {
				return CompareInt(int64(v), pattern)
			}
		}
		return false
	}, nil
}

func (i *Int) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfDouble:
		return &Double{float64(i.Native)}
	case *TypeOfInt:
		return i
	case *TypeOfCount:
		return &Count{uint64(i.Native)}
	case *TypeOfPort:
		return &Port{uint32(i.Native)}
	case *TypeOfTime:
		return &Time{nano.Ts(i.Native)}
	case *TypeOfInterval:
		return &Interval{i.Native}
	}
	return nil
}

// CoerceToInt attempts to convert a value to an integer and returns a new
// integer value.  Int, Count, and Port can are all translated to an Int
// with the same native value while a Double is converted only if the
// double is an integer.  Time and Intervals are converted to an Int as
// their nanosecond values.  If the value cannot be coerced, then nil is
// returned.
func CoerceToInt(in Value) *Int {
	switch v := in.(type) {
	case *Int:
		return v
	case *Count:
		return &Int{int64(v.Native)}
	case *Port:
		return &Int{int64(v.Native)}
	case *Double:
		i := int64(v.Native)
		if float64(i) == v.Native {
			return &Int{i}
		}
	case *Time:
		return &Int{int64(v.Native)}
	case *Interval:
		return &Int{int64(v.Native)}
	}
	return nil
}

func (i *Int) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Native)
}

func (i *Int) Elements() ([]Value, bool) { return nil, false }
