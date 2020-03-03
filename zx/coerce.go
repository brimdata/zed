package zx

import (
	"fmt"
	"math"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

// CoerceToFloat64 attempts to convert a value to a float64. The
// resulting coerced value is written to out, and true is returned. If
// the value cannot be coerced, then false is returned.
func CoerceToFloat64(in zng.Value) (float64, bool) {
	var out float64
	var err error
	switch in.Type.ID() {
	default:
		return 0, false
	case zng.IdFloat64:
		out, err = zng.DecodeFloat64(in.Bytes)
	case zng.IdByte:
		var b byte
		b, err = zng.DecodeByte(in.Bytes)
		out = float64(b)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		var v int64
		v, err = zng.DecodeInt(in.Bytes)
		out = float64(v)
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		var v uint64
		v, err = zng.DecodeUint(in.Bytes)
		out = float64(v)
	case zng.IdBool:
		var v bool
		v, err = zng.DecodeBool(in.Bytes)
		if v {
			out = 1
		}
	case zng.IdPort:
		var v uint32
		v, err = zng.DecodePort(in.Bytes)
		out = float64(v)
	case zng.IdTime:
		var v nano.Ts
		v, err = zng.DecodeTime(in.Bytes)
		out = float64(v) / 1e9
	case zng.IdDuration:
		var v int64
		v, err = zng.DecodeDuration(in.Bytes)
		out = float64(v) / 1e9
	}
	if err != nil {
		return 0, false
	}
	return out, true
}

// CoerceToInt and CoerceToUint attempt to convert a value to the given
// integer type.  Integer types (byte, int*, uint*) can be coerced
// to another integer type if the value is in range.  float64 may
// be coerced to an integer type only if the value is equivalent.
// Time and Intervals are converted to an Int as their nanosecond
// values.
func CoerceToInt(in zng.Value, typ zng.Type) (zng.Value, bool) {
	var i int64
	var err error

	switch in.Type.ID() {
	case zng.IdFloat64:
		var v float64
		v, err = zng.DecodeFloat64(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		i = int64(v)
		if float64(i) != v {
			return zng.Value{}, false
		}
	case zng.IdTime:
		var v nano.Ts
		v, err = zng.DecodeTime(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		i = int64(v / 1e9)
	case zng.IdDuration:
		var v int64
		v, err = zng.DecodeDuration(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		i = int64(v / 1e9)
	case zng.IdByte:
		var b byte
		b, err = zng.DecodeByte(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		i = int64(b)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		i, err = zng.DecodeInt(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		u, err := zng.DecodeUint(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		// Further checking on the desired type happens below but
		// first make sure this fits in a signed int64 type.
		if u > math.MaxInt64 {
			return zng.Value{}, false
		}
		i = int64(u)
	default:
		// can't be cast to integer
		return zng.Value{}, false
	}

	switch typ.ID() {
	case zng.IdInt16:
		if i < math.MinInt16 || i > math.MaxInt16 {
			return zng.Value{}, false
		}
	case zng.IdInt32:
		if i < math.MinInt32 || i > math.MaxInt32 {
			return zng.Value{}, false
		}
	case zng.IdInt64:
		// it already fits, no checking needed

	default:
		panic(fmt.Sprintf("Called CoerceToInt on non-integer type %s", typ))
	}

	return zng.Value{typ, zng.EncodeInt(i)}, true
}

func CoerceToUint(in zng.Value, typ zng.Type) (zng.Value, bool) {
	var i uint64
	var err error

	switch in.Type.ID() {
	case zng.IdFloat64:
		var v float64
		v, err = zng.DecodeFloat64(in.Bytes)
		i = uint64(v)
		if float64(i) != v {
			return zng.Value{}, false
		}
	case zng.IdTime:
		var v nano.Ts
		v, err = zng.DecodeTime(in.Bytes)
		i = uint64(v / 1e9)
	case zng.IdDuration:
		var v int64
		v, err = zng.DecodeDuration(in.Bytes)
		i = uint64(v / 1e9)
	case zng.IdByte:
		var b byte
		b, err = zng.DecodeByte(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		i = uint64(b)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		var si int64
		si, err = zng.DecodeInt(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
		// Further checking on the desired type happens below but
		// first make sure the value isn't negative.
		if si < 0 {
			return zng.Value{}, false
		}
		i = uint64(si)
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		i, err = zng.DecodeUint(in.Bytes)
		if err != nil {
			return zng.Value{}, false
		}
	default:
		// can't be cast to integer
		return zng.Value{}, false
	}

	switch typ.ID() {
	case zng.IdUint16:
		if i > math.MaxUint16 {
			return zng.Value{}, false
		}
	case zng.IdUint32:
		if i > math.MaxUint32 {
			return zng.Value{}, false
		}
	case zng.IdUint64:
		// it already fits, no checking needed

	default:
		panic(fmt.Sprintf("Called CoerceToInt on non-integer type %s", typ))
	}

	return zng.Value{typ, zng.EncodeUint(i)}, true
}

// CoerceToDuration attempts to convert a value to a duration.  Int
// and Double are converted as seconds. The resulting coerced value is
// written to out, and true is returned. If the value cannot be
// coerced, then false is returned.
func CoerceToDuration(in zng.Value) (int64, bool) {
	var out int64
	var err error
	switch in.Type.ID() {
	default:
		return 0, false
	case zng.IdDuration:
		out, err = zng.DecodeDuration(in.Bytes)
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		var v uint64
		v, err = zng.DecodeUint(in.Bytes)
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		out = 1_000_000_000 * int64(v)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		out, err = zng.DecodeInt(in.Bytes)
		out *= 1_000_000_000
	case zng.IdFloat64:
		var v float64
		v, err = zng.DecodeFloat64(in.Bytes)
		v *= 1e9
		out = int64(v)
	}
	if err != nil {
		return 0, false
	}
	return out, true
}

func CoerceToPort(in zng.Value) (uint32, bool) {
	var out uint32
	var err error
	body := in.Bytes
	switch in.Type.ID() {
	default:
		return 0, false
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		var v int64
		v, err = zng.DecodeInt(body)
		// check for overflow
		if v < 0 || v > math.MaxUint16 {
			return 0, false
		}
		out = uint32(v)
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		var v uint64
		v, err = zng.DecodeUint(body)
		// check for overflow
		if v > math.MaxUint16 {
			return 0, false
		}
		out = uint32(v)
	case zng.IdPort:
		out, err = zng.DecodePort(body)
	case zng.IdFloat64:
		var v float64
		v, err = zng.DecodeFloat64(body)
		out = uint32(v)
		if float64(out) != v {
			return 0, false
		}
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
	switch in.Type.ID() {
	default:
		return 0, false
	case zng.IdTime:
		ts, err = zng.DecodeTime(in.Bytes)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		var v int64
		v, err = zng.DecodeInt(in.Bytes)
		ts = nano.Ts(v) * 1_000_000_000
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		var v uint64
		v, err = zng.DecodeUint(in.Bytes)
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		ts = nano.Ts(v)
	case zng.IdFloat64:
		var v float64
		v, err = zng.DecodeFloat64(in.Bytes)
		ts = nano.Ts(v * 1e9)
	}
	if err != nil {
		return 0, false
	}
	return ts, true
}

func CoerceToString(in zng.Value) (string, bool) {
	switch in.Type.ID() {
	default:
		return "", false
	case zng.IdString, zng.IdBstring, zng.IdEnum:
		return string(in.Bytes), true
	}
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
	switch to.ID() {
	case zng.IdFloat64:
		if d, ok := CoerceToFloat64(v); ok {
			return zng.NewFloat64(d), true
		}
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		return CoerceToInt(v, to)
	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		return CoerceToUint(v, to)
	case zng.IdDuration:
		if i, ok := CoerceToDuration(v); ok {
			return zng.NewDuration(i), true
		}
	case zng.IdPort:
		if p, ok := CoerceToPort(v); ok {
			return zng.NewPort(p), true
		}
	case zng.IdTime:
		if i, ok := CoerceToTime(v); ok {
			return zng.NewTime(i), true
		}
	case zng.IdString:
		if s, ok := CoerceToString(v); ok {
			return zng.NewString(s), true
		}
	case zng.IdBstring:
		if s, ok := CoerceToString(v); ok {
			return zng.NewBstring(s), true
		}
	}
	return zng.Value{}, false
}
