package zx

import (
	"math"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zng"
)

// CoerceToDouble attempts to convert a value to a double. The
// resulting coerced value is written to out, and true is returned. If
// the value cannot be coerced, then false is returned.
func CoerceToDouble(in zng.Value) (float64, bool) {
	var out float64
	var err error
	switch in.Type.(type) {
	default:
		return 0, false
	case *zng.TypeOfDouble:
		out, err = zng.DecodeDouble(in.Bytes)
	case *zng.TypeOfInt:
		var v int64
		v, err = zng.DecodeInt(in.Bytes)
		out = float64(v)
	case *zng.TypeOfBool:
		var v bool
		v, err = zng.DecodeBool(in.Bytes)
		if v {
			out = 1
		}
	case *zng.TypeOfCount:
		var v uint64
		v, err = zng.DecodeCount(in.Bytes)
		out = float64(v)
	case *zng.TypeOfPort:
		var v uint32
		v, err = zng.DecodePort(in.Bytes)
		out = float64(v)
	case *zng.TypeOfTime:
		var v nano.Ts
		v, err = zng.DecodeTime(in.Bytes)
		out = float64(v) / 1e9
	case *zng.TypeOfInterval:
		var v int64
		v, err = zng.DecodeInterval(in.Bytes)
		out = float64(v) / 1e9
	}
	if err != nil {
		return 0, false
	}
	return out, true
}

// CoerceToInt attempts to convert a value to an integer.  Int, Count,
// and Port can are all translated to an Int with the same native
// value while a Double is converted only if the double is an integer.
// Time and Intervals are converted to an Int as their nanosecond
// values. The resulting coerced value is written to out, and true is
// returned. If the value cannot be coerced, then false is returned.
func CoerceToInt(in zng.Value) (int64, bool) {
	var out int64
	var err error
	body := in.Bytes
	switch in.Type.(type) {
	default:
		return 0, false
	case *zng.TypeOfInt:
		out, err = zng.DecodeInt(body)
	case *zng.TypeOfCount:
		var v uint64
		v, err = zng.DecodeCount(body)
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		out = int64(v)
	case *zng.TypeOfPort:
		var v uint32
		v, err = zng.DecodePort(body)
		out = int64(v)
	case *zng.TypeOfDouble:
		var v float64
		v, err = zng.DecodeDouble(body)
		out = int64(v)
		if float64(out) != v {
			return 0, false
		}
	case *zng.TypeOfTime:
		var v nano.Ts
		v, err = zng.DecodeTime(body)
		out = int64(v / 1e9)
	case *zng.TypeOfInterval:
		var v int64
		v, err = zng.DecodeInterval(body)
		out = int64(v / 1e9)
	}
	if err != nil {
		return 0, false
	}
	return out, true
}

// CoerceToInterval attempts to convert a value to an interval.  Int
// and Double are converted as seconds. The resulting coerced value is
// written to out, and true is returned. If the value cannot be
// coerced, then false is returned.
func CoerceToInterval(in zng.Value) (int64, bool) {
	var out int64
	var err error
	switch in.Type.(type) {
	default:
		return 0, false
	case *zng.TypeOfInterval:
		out, err = zng.DecodeInterval(in.Bytes)
	case *zng.TypeOfInt:
		out, err = zng.DecodeInt(in.Bytes)
		out *= 1_000_000_000
	case *zng.TypeOfDouble:
		var v float64
		v, err = zng.DecodeDouble(in.Bytes)
		v *= 1e9
		out = int64(v)
	}
	if err != nil {
		return 0, false
	}
	return out, true
}

// CoerceToTime attempts to convert a value to a time. Int and Double
// are converted as seconds. The resulting coerced value is written to
// out, and true is returned. If the value cannot be coerced, then
// false is returned.
func CoerceToTime(in zng.Value) (nano.Ts, bool) {
	var err error
	var ts nano.Ts
	switch in.Type.(type) {
	default:
		return 0, false
	case *zng.TypeOfTime:
		ts, err = zng.DecodeTime(in.Bytes)
	case *zng.TypeOfInt:
		var v int64
		v, err = zng.DecodeInt(in.Bytes)
		ts = nano.Ts(v) * 1_000_000_000
	case *zng.TypeOfDouble:
		var v float64
		v, err = zng.DecodeDouble(in.Bytes)
		ts = nano.Ts(v * 1e9)
	}
	if err != nil {
		return 0, false
	}
	return ts, true
}

// TBD
// Coerce tries to convert this value to an equal value of a different
// type.  For example, calling Coerce(TypeDouble) on a value that is
// an int(100) will return a double(100.0).
// XXX this doesn't seem valid:  If the coercion cannot be
// performed such that v.Coerce(t1).Coerce(v.Type).String() == v.String(),
// then nil is returned.
func Coerce(v zng.Value, to zng.Type) (zng.Value, bool) {
	if v.Type == to {
		return v, true
	}
	switch to.(type) {
	case *zng.TypeOfDouble:
		if d, ok := CoerceToDouble(v); ok {
			return zng.NewDouble(d), true
		}
	case *zng.TypeOfInt:
		if i, ok := CoerceToInt(v); ok {
			return zng.NewInt(i), true
		}
	case *zng.TypeOfInterval:
		if i, ok := CoerceToInterval(v); ok {
			return zng.NewInterval(i), true
		}
	case *zng.TypeOfTime:
		if i, ok := CoerceToTime(v); ok {
			return zng.NewTime(i), true
		}
	}
	return zng.Value{}, false
}
