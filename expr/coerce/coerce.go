package coerce

import (
	"bytes"
	"errors"
	"math"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

var ErrOverflow = errors.New("integer overflow: uint64 value too large for int64")
var ErrIncompatibleTypes = errors.New("incompatible types")

// XXX aliases should probably be preserved according to the rank
// of the underlying number type.

// Pair provides a buffer to decode values into while doing comparisons
// so the same buffers can be reused on each call without zcode.Bytes buffers
// escaping to GC.  This method uses the zed.AppendInt(), zed.AppendUint(),
// etc to encode zcode.Bytes as an in-place slice instead of allocating
// new slice buffers for every value created.
type Pair struct {
	// a and b point to inputs that can't change
	A zcode.Bytes
	B zcode.Bytes
	// Buffer is a scratch buffer that stays around between calls and is the
	// landing place for either the a or b value if one of them needs to
	// be coerced (you never need to coerce both).  Then we point a or b
	// at buf and let go of the other input pointer.
	result.Buffer
}

func (c *Pair) Equal() bool {
	// bytes.Equal() returns true for nil compared to an empty-slice,
	// which doesn't work for Zed null comparisons, so we explicitly check
	// for the nil condition here.
	if c.A == nil {
		return c.B == nil
	}
	if c.B == nil {
		return c.A == nil
	}
	return bytes.Equal(c.A, c.B)
}

func (c *Pair) Coerce(a, b zed.Value) (int, error) {
	c.A = a.Bytes
	c.B = b.Bytes
	if a.Type == nil {
		a.Type = zed.TypeNull
	}
	if b.Type == nil {
		b.Type = zed.TypeNull
	}
	aid := a.Type.ID()
	bid := b.Type.ID()
	if aid == bid {
		return aid, nil
	}
	if aid == zed.IDNull {
		return bid, nil
	}
	if bid == zed.IDNull {
		return aid, nil
	}
	if zed.IsNumber(aid) {
		if !zed.IsNumber(bid) {
			return 0, ErrIncompatibleTypes
		}
		return c.coerceNumbers(aid, bid)
	}
	if zed.IsStringy(aid) && zed.IsStringy(bid) {
		// Promote to bstring if they are different
		id := aid
		if id != bid {
			id = zed.IDBstring
		}
		return id, nil
	}
	return 0, ErrIncompatibleTypes
}

func (c *Pair) compare(lhs, rhs zed.Value) (bool, error) {
	if _, err := c.Coerce(lhs, rhs); err != nil {
		return false, err
	}
	return c.Equal(), nil
}

func intToFloat(id int, b zcode.Bytes) float64 {
	if zed.IsSigned(id) {
		v, _ := zed.DecodeInt(b)
		return float64(v)
	}
	v, _ := zed.DecodeUint(b)
	return float64(v)
}

func (c *Pair) promoteToSigned(in zcode.Bytes) (zcode.Bytes, error) {
	v, _ := zed.DecodeUint(in)
	if v > math.MaxInt64 {
		return nil, ErrOverflow
	}
	return c.Int(int64(v)), nil
}

func (c *Pair) promoteToUnsigned(in zcode.Bytes) (zcode.Bytes, error) {
	v, _ := zed.DecodeInt(in)
	if v < 0 {
		return nil, ErrOverflow
	}
	return c.Uint(uint64(v)), nil
}

func (c *Pair) coerceNumbers(aid, bid int) (int, error) {
	if zed.IsFloat(aid) {
		c.B = c.Float64(intToFloat(bid, c.B))
		return aid, nil
	}
	if zed.IsFloat(bid) {
		c.A = c.Float64(intToFloat(aid, c.A))
		return bid, nil
	}
	aIsSigned := zed.IsSigned(aid)
	if aIsSigned == zed.IsSigned(bid) {
		// They have the same signed-ness.  Promote to the wider
		// type by rank and leave the zcode.Bytes as is since
		// the varint encoding is the same for all the widths.
		// Width increasese with type ID.
		id := aid
		if bid > id {
			id = bid
		}
		return id, nil
	}
	id := zed.PromoteInt(aid, bid)

	// Otherwise, we'll promote mixed signed-ness to signed unless
	// the unsigned value is greater than signed maxint, in which
	// case, we report an overflow error.
	var err error
	if aIsSigned {
		c.B, err = c.promoteToSigned(c.B)
	} else {
		c.A, err = c.promoteToSigned(c.A)
	}
	if err == ErrOverflow {
		// We got overflow trying to turn the unsigned to signed,
		// so try turning the signed into unsigned.
		if aIsSigned {
			c.A, err = c.promoteToUnsigned(c.A)
		} else {
			c.B, err = c.promoteToUnsigned(c.B)
		}
		id = zed.IDUint64
	}
	return id, err
}

func ToFloat(zv zed.Value) (float64, bool) {
	id := zv.Type.ID()
	if zed.IsFloat(id) {
		f, _ := zed.DecodeFloat64(zv.Bytes)
		return f, true
	}
	if zed.IsInteger(id) {
		if zed.IsSigned(id) {
			v, _ := zed.DecodeInt(zv.Bytes)
			return float64(v), true
		} else {
			v, _ := zed.DecodeUint(zv.Bytes)
			return float64(v), true
		}
	}
	if id == zed.IDDuration {
		v, _ := zed.DecodeInt(zv.Bytes)
		return 1e-9 * float64(v), true
	}
	if zed.IsStringy(id) {
		v, err := strconv.ParseFloat(string(zv.Bytes), 64)
		return v, err == nil
	}
	return 0, false
}

func ToUint(zv zed.Value) (uint64, bool) {
	id := zv.Type.ID()
	if zed.IsFloat(id) {
		f, _ := zed.DecodeFloat64(zv.Bytes)
		return uint64(f), true
	}
	if zed.IsInteger(id) {
		if zed.IsSigned(id) {
			v, _ := zed.DecodeInt(zv.Bytes)
			if v < 0 {
				return 0, false
			}
			return uint64(v), true
		} else {
			v, _ := zed.DecodeUint(zv.Bytes)
			return uint64(v), true
		}
	}
	if id == zed.IDDuration {
		v, _ := zed.DecodeInt(zv.Bytes)
		return uint64(v / 1_000_000_000), true
	}
	if zed.IsStringy(id) {
		v, err := strconv.ParseUint(string(zv.Bytes), 10, 64)
		return v, err == nil
	}
	return 0, false
}

func ToInt(zv zed.Value) (int64, bool) {
	id := zv.Type.ID()
	if zed.IsFloat(id) {
		f, _ := zed.DecodeFloat64(zv.Bytes)
		return int64(f), true
	}
	if zed.IsInteger(id) {
		if zed.IsSigned(id) {
			v, _ := zed.DecodeInt(zv.Bytes)
			// XXX check if negative? should -1:uint64 be maxint64 or an error?
			return int64(v), true
		} else {
			v, _ := zed.DecodeUint(zv.Bytes)
			return int64(v), true
		}
	}
	if id == zed.IDDuration {
		v, _ := zed.DecodeInt(zv.Bytes)
		return int64(v / 1_000_000_000), true
	}
	if zed.IsStringy(id) {
		v, err := strconv.ParseInt(string(zv.Bytes), 10, 64)
		return v, err == nil
	}
	return 0, false
}

func ToBool(zv zed.Value) (bool, bool) {
	if zv.IsStringy() {
		v, err := strconv.ParseBool(string(zv.Bytes))
		return v, err == nil
	}
	v, ok := ToInt(zv)
	return v != 0, ok
}

func ToTime(zv zed.Value) (nano.Ts, bool) {
	id := zv.Type.ID()
	if id == zed.IDTime {
		ts, _ := zed.DecodeTime(zv.Bytes)
		return ts, true
	}
	if zed.IsSigned(id) {
		v, _ := zed.DecodeInt(zv.Bytes)
		return nano.Ts(v) * 1_000_000_000, true
	}
	if zed.IsInteger(id) {
		v, _ := zed.DecodeUint(zv.Bytes)
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		return nano.Ts(v), true
	}
	if zed.IsFloat(id) {
		v, _ := zed.DecodeFloat64(zv.Bytes)
		return nano.Ts(v * 1e9), true
	}
	return 0, false
}

// ToDuration attempts to convert a value to a duration.  Int
// and Double are converted as seconds. The resulting coerced value is
// written to out, and true is returned. If the value cannot be
// coerced, then false is returned.
func ToDuration(in zed.Value) (nano.Duration, bool) {
	var out nano.Duration
	var err error
	switch in.Type.ID() {
	case zed.IDDuration:
		out, err = zed.DecodeDuration(in.Bytes)
	case zed.IDUint16, zed.IDUint32, zed.IDUint64:
		var v uint64
		v, err = zed.DecodeUint(in.Bytes)
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		out = nano.Duration(v) * nano.Second
	case zed.IDInt16, zed.IDInt32, zed.IDInt64:
		var v int64
		v, err = zed.DecodeInt(in.Bytes)
		//XXX check for overflow here
		out = nano.Duration(v) * nano.Second
	case zed.IDFloat64:
		var v float64
		v, err = zed.DecodeFloat64(in.Bytes)
		out = nano.DurationFromFloat(v)
	default:
		return 0, false
	}
	if err != nil {
		return 0, false
	}
	return out, true
}
