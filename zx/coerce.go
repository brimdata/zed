package zx

import (
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

// CoerceToInt attempts to convert a value to a signed integer.
// This always succeeds for signed integer types and succeeds for
// unsigned integers so long as they are not greater than MaxInt64.
// A float64 may be coerced only if the value is equivalent.
// Time and Intervals are converted as their nanosecond values.
func CoerceToInt(in Value) (int64, bool) {
	switch in.Type.ID() {
	case IdFloat64:
		v, err := DecodeFloat64(in.Bytes)
		if err != nil {
			return 0, false
		}
		i := int64(v)
		if float64(i) != v {
			return 0, false
		}
		return i, true
	case IdTime:
		v, err := DecodeTime(in.Bytes)
		if err != nil {
			return 0, false
		}
		return int64(v / 1e9), true
	case IdDuration:
		v, err := DecodeDuration(in.Bytes)
		if err != nil {
			return 0, false
		}
		return int64(v / 1e9), true
	case IdByte:
		b, err := DecodeByte(in.Bytes)
		if err != nil {
			return 0, false
		}
		return int64(b), true
	case IdInt16, IdInt32, IdInt64:
		i, err := DecodeInt(in.Bytes)
		if err != nil {
			return 0, false
		}
		return i, true
	case IdUint16, IdUint32, IdUint64:
		u, err := DecodeUint(in.Bytes)
		if err != nil {
			return 0, false
		}
		// Further checking on the desired type happens below but
		// first make sure this fits in a signed int64 type.
		if u > math.MaxInt64 {
			return 0, false
		}
		return int64(u), true
	default:
		// can't be cast to integer
		return 0, false
	}
}

// CoerceToUint attempts to convert a value to an unsigned integer.
// This always succeeds for unsigned integer types and succeeds for
// signed integers so long as they are not negative.
// A float64 may be coerced only if the value is equivalent.
// Time and Intervals are converted as their nanosecond values.
func CoerceToUint(in Value) (uint64, bool) {
	switch in.Type.ID() {
	case IdFloat64:
		v, err := DecodeFloat64(in.Bytes)
		if err != nil {
			return 0, false
		}
		i := uint64(v)
		if float64(i) != v {
			return 0, false
		}
		return i, true
	case IdTime:
		v, err := DecodeTime(in.Bytes)
		if err != nil {
			return 0, false
		}
		return uint64(v / 1e9), true
	case IdDuration:
		v, err := DecodeDuration(in.Bytes)
		if err != nil {
			return 0, false
		}
		return uint64(v / 1e9), true
	case IdByte:
		b, err := DecodeByte(in.Bytes)
		if err != nil {
			return 0, false
		}
		return uint64(b), true
	case IdInt16, IdInt32, IdInt64:
		si, err := DecodeInt(in.Bytes)
		if err != nil {
			return 0, false
		}
		if si < 0 {
			return 0, false
		}
		return uint64(si), true
	case IdUint16, IdUint32, IdUint64:
		i, err := DecodeUint(in.Bytes)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		// can't be cast to unsigned integer
		return 0, false
	}
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
