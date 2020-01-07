package zng

import (
	"fmt"
	"math"
	"strconv"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
)

type TypeOfInt struct{}

var compareInt = map[string]func(int64, int64) bool{
	"eql":  func(a, b int64) bool { return a == b },
	"neql": func(a, b int64) bool { return a != b },
	"gt":   func(a, b int64) bool { return a > b },
	"gte":  func(a, b int64) bool { return a >= b },
	"lt":   func(a, b int64) bool { return a < b },
	"lte":  func(a, b int64) bool { return a <= b }}

func (t *TypeOfInt) String() string {
	return "int"
}

func EncodeInt(i int64) zcode.Bytes {
	var b [8]byte
	n := encodeInt(b[:], i)
	return b[:n]
}

func DecodeInt(zv zcode.Bytes) (int64, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return decodeInt(zv), nil
}

func (t *TypeOfInt) Parse(in []byte) (zcode.Bytes, error) {
	i, err := UnsafeParseInt64(in)
	if err != nil {
		return nil, err
	}
	return EncodeInt(i), nil
}

func (t *TypeOfInt) New(zv zcode.Bytes) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	v, err := DecodeInt(zv)
	if err != nil {
		return nil, err
	}
	return NewInt(v), nil
}

type Int int64

func NewInt(i int64) *Int {
	p := Int(i)
	return &p
}

func (i Int) String() string {
	return strconv.FormatInt(int64(i), 10)
}

func (i Int) Encode(dst zcode.Bytes) zcode.Bytes {
	return zcode.AppendValue(dst, EncodeInt(int64(i)))
}

func (i Int) Type() Type {
	return TypeInt
}

func (i Int) NativeComparison(op string) (func(int64) bool, error) {
	compare, ok := compareInt[op]
	if !ok {
		return nil, fmt.Errorf("unknown comparator: %s", op)
	}
	return func(val int64) bool {
		return compare(val, int64(i))
	}, nil
}

// Comparison returns a Predicate that compares typed byte slices that must
// be coercible to an integer with the value's integer value using a comparison
// based on op.  Int, count, port, double, bool, time and interval types can
// all be converted to the integer value.  XXX there are some overflow issues here.
func (i Int) Comparison(op string) (Predicate, error) {
	CompareInt, ok1 := compareInt[op]
	CompareFloat, ok2 := compareFloat[op]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown int comparator: %s", op)
	}
	pattern := int64(i)
	// many different zeek data types can be compared with integers
	return func(e TypedEncoding) bool {
		val := e.Body
		switch e.Type.(type) {
		case *TypeOfInt:
			// we can parse counts and ports as an integer
			v, err := DecodeInt(val)
			if err == nil {
				return CompareInt(v, pattern)
			}
		case *TypeOfCount:
			// we can parse counts and ports as an integer
			v, err := DecodeCount(val)
			if err == nil {
				return CompareInt(int64(v), pattern)
			}
		case *TypeOfPort:
			// we can parse counts and ports as an integer
			v, err := DecodePort(val)
			if err == nil {
				return CompareInt(int64(v), pattern)
			}
		case *TypeOfDouble:
			v, err := DecodeDouble(val)
			if err == nil {
				return CompareFloat(v, float64(pattern))
			}
		case *TypeOfTime:
			ts, err := DecodeTime(val)
			if err == nil {
				return CompareInt(int64(ts), pattern*1e9)
			}
		case *TypeOfInterval:
			v, err := DecodeInt(val)
			if err == nil {
				return CompareInt(int64(v), pattern*1e9)
			}
		}
		return false
	}, nil
}

func (i Int) Coerce(typ Type) Value {
	switch typ.(type) {
	case *TypeOfDouble:
		return NewDouble(float64(i))
	case *TypeOfInt:
		return i
	case *TypeOfCount:
		return NewCount(uint64(i))
	case *TypeOfPort:
		return NewPort(uint32(i))
	case *TypeOfTime:
		return NewTime(nano.Ts(i * 1e9))
	case *TypeOfInterval:
		return NewInterval(int64(i * 1e9))
	}
	return nil
}

// CoerceToInt attempts to convert a value to an integer.  Int, Count,
// and Port can are all translated to an Int with the same native
// value while a Double is converted only if the double is an integer.
// Time and Intervals are converted to an Int as their nanosecond
// values. The resulting coerced value is written to out, and true is
// returned. If the value cannot be coerced, then false is returned.
func CoerceToInt(in Value, out *Int) bool {
	switch v := in.(type) {
	case *Int:
		*out = *v
		return true
	case *Count:
		u := uint64(*v)
		// check for overflow
		if u > math.MaxInt64 {
			return false
		}
		*out = Int(int64(*v))
		return true
	case *Port:
		*out = Int(int64(*v))
		return true
	case *Double:
		i := int64(*v)
		if float64(i) == float64(*v) {
			*out = Int(i)
			return true
		}
	case *Time:
		*out = Int(int64(*v))
		return true
	case *Interval:
		*out = Int(*v)
		return true
	}
	return false
}

func (i Int) Elements() ([]Value, bool) { return nil, false }
