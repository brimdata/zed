package zeek

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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

func EncodeDouble(d float64) zval.Encoding {
	bits := math.Float64bits(d)
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], bits)
	return b[:]
}

func DecodeDouble(zv zval.Encoding) (float64, error) {
	if len(zv) != 8 {
		return 0, errors.New("byte encoding of double not 8 bytes")
	}
	bits := binary.LittleEndian.Uint64(zv)
	return math.Float64frombits(bits), nil
}

func (t *TypeOfDouble) Parse(in []byte) (zval.Encoding, error) {
	d, err := UnsafeParseFloat64(in)
	if err != nil {
		return nil, err
	}
	return EncodeDouble(d), nil
}

func (t *TypeOfDouble) New(zv zval.Encoding) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	f, err := DecodeDouble(zv)
	if err != nil {
		return nil, err
	}
	return NewDouble(f), nil
}

type Double float64

func NewDouble(f float64) *Double {
	d := Double(f)
	return &d
}

func (d Double) String() string {
	return strconv.FormatFloat(float64(d), 'f', -1, 64)
}

func (d Double) Encode(dst zval.Encoding) zval.Encoding {
	return zval.AppendValue(dst, EncodeDouble(float64(d)))
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
		switch e.Type.(type) {
		// We allow comparison of float constant with integer-y
		// fields and just use typeDouble to parse since it will do
		// the right thing for integers.  XXX do we want to allow
		// integers that cause float64 overflow?  user can always
		// use an integer constant instead of a float constant to
		// compare with the integer-y field.
		case *TypeOfDouble:
			v, err := DecodeDouble(val)
			if err == nil {
				return compare(v, pattern)
			}
		case *TypeOfInt:
			v, err := DecodeInt(val)
			if err == nil {
				return compare(float64(v), pattern)
			}
		case *TypeOfCount:
			v, err := DecodeCount(val)
			if err == nil {
				return compare(float64(v), pattern)
			}
		case *TypeOfPort:
			v, err := DecodePort(val)
			if err == nil {
				return compare(float64(v), pattern)
			}

		case *TypeOfTime:
			ts, err := DecodeTime(val)
			if err == nil {
				return compare(float64(ts)/1e9, pattern)
			}
		case *TypeOfInterval:
			v, err := DecodeInt(val)
			if err == nil {
				return compare(float64(v)/1e9, pattern)
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

// CoerceToDouble attempts to convert a value to a double. The
// resulting coerced value is written to out, and true is returned. If
// the value cannot be coerced, then false is returned.
func CoerceToDouble(in Value, out *Double) bool {
	switch v := in.(type) {
	case *Double:
		*out = *v
		return true
	case *Int:
		*out = Double(float64(*v))
		return true
	case *Bool:
		if *v {
			*out = Double(1)
		} else {
			*out = Double(0)
		}
		return true
	case *Count:
		*out = Double(float64(*v))
		return true
	case *Port:
		*out = Double(float64(*v))
		return true
	case *Time:
		*out = Double(float64(*v) / 1e9)
		return true
	case *Interval:
		*out = Double(float64(*v) / 1e9)
		return true
	}
	return false
}

func (d Double) Elements() ([]Value, bool) { return nil, false }
