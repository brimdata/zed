package coerce

import (
	"bytes"
	"errors"
	"math"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/result"
	"github.com/brimdata/zed/zcode"
)

var Overflow = errors.New("integer overflow: uint64 value too large for int64")
var IncompatibleTypes = errors.New("incompatible types")

// XXX Named types should probably be preserved according to the rank
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
	buf2 result.Buffer
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

func (c *Pair) Coerce(a, b *zed.Value) (int, error) {
	c.A = a.Bytes()
	c.B = b.Bytes()
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
			return 0, IncompatibleTypes
		}
		id, ok := c.coerceNumbers(aid, bid)
		if !ok {
			return 0, Overflow
		}
		return id, nil
	}
	return 0, IncompatibleTypes
}

func intToFloat(id int, b zcode.Bytes) float64 {
	if zed.IsSigned(id) {
		return float64(zed.DecodeInt(b))
	}
	return float64(zed.DecodeUint(b))
}

func (c *Pair) promoteToSigned(in zcode.Bytes) (zcode.Bytes, bool) {
	v := zed.DecodeUint(in)
	if v > math.MaxInt64 {
		return nil, false
	}
	return c.Int(int64(v)), true
}

func (c *Pair) promoteToUnsigned(in zcode.Bytes) (zcode.Bytes, bool) {
	v := zed.DecodeInt(in)
	if v < 0 {
		return nil, false
	}
	return c.Uint(uint64(v)), true
}

func (c *Pair) coerceNumbers(aid, bid int) (int, bool) {
	if zed.IsFloat(aid) {
		if aid == zed.IDFloat16 {
			c.A = c.buf2.Float64(float64(zed.DecodeFloat16(c.A)))
		} else if aid == zed.IDFloat32 {
			c.A = c.buf2.Float64(float64(zed.DecodeFloat32(c.A)))
		}
		c.B = c.Float64(intToFloat(bid, c.B))
		return aid, true
	}
	if zed.IsFloat(bid) {
		if bid == zed.IDFloat16 {
			c.B = c.buf2.Float64(float64(zed.DecodeFloat16(c.B)))
		} else if bid == zed.IDFloat32 {
			c.B = c.buf2.Float64(float64(zed.DecodeFloat32(c.B)))
		}
		c.A = c.Float64(intToFloat(aid, c.A))
		return bid, true
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
		return id, true
	}
	id := promoteInt(aid, bid)

	// Otherwise, we'll promote mixed signed-ness to signed unless
	// the unsigned value is greater than signed maxint, in which
	// case, we report an overflow error.
	var ok bool
	if aIsSigned {
		c.B, ok = c.promoteToSigned(c.B)
	} else {
		c.A, ok = c.promoteToSigned(c.A)
	}
	if !ok {
		// We got overflow trying to turn the unsigned to signed,
		// so try turning the signed into unsigned.
		if aIsSigned {
			c.A, ok = c.promoteToUnsigned(c.A)
		} else {
			c.B, ok = c.promoteToUnsigned(c.B)
		}
		id = zed.IDUint64
	}
	return id, ok
}

func ToFloat(val *zed.Value) (float64, bool) {
	id := val.Type.ID()
	if zed.IsFloat(id) {
		return zed.DecodeFloat(val.Bytes()), true
	}
	if zed.IsInteger(id) {
		if zed.IsSigned(id) {
			return float64(zed.DecodeInt(val.Bytes())), true
		} else {
			return float64(zed.DecodeUint(val.Bytes())), true
		}
	}
	if id == zed.IDDuration {
		return float64(zed.DecodeInt(val.Bytes())), true
	}
	if id == zed.IDTime {
		return float64(zed.DecodeTime(val.Bytes())), true
	}
	if id == zed.IDString {
		v, err := strconv.ParseFloat(string(val.Bytes()), 64)
		return v, err == nil
	}
	return 0, false
}

func ToUint(val *zed.Value) (uint64, bool) {
	id := val.Type.ID()
	if zed.IsFloat(id) {
		return uint64(zed.DecodeFloat(val.Bytes())), true
	}
	if zed.IsInteger(id) {
		if zed.IsSigned(id) {
			v := zed.DecodeInt(val.Bytes())
			if v < 0 {
				return 0, false
			}
			return uint64(v), true
		} else {
			return zed.DecodeUint(val.Bytes()), true
		}
	}
	if id == zed.IDDuration {
		return uint64(zed.DecodeInt(val.Bytes())), true
	}
	if id == zed.IDTime {
		return uint64(zed.DecodeTime(val.Bytes())), true
	}
	if id == zed.IDString {
		v, err := strconv.ParseUint(string(val.Bytes()), 10, 64)
		return v, err == nil
	}
	return 0, false
}

func ToInt(val *zed.Value) (int64, bool) {
	id := val.Type.ID()
	if zed.IsFloat(id) {
		return int64(zed.DecodeFloat(val.Bytes())), true
	}
	if zed.IsInteger(id) {
		if zed.IsSigned(id) {
			// XXX check if negative? should -1:uint64 be maxint64 or an error?
			return zed.DecodeInt(val.Bytes()), true
		} else {
			return int64(zed.DecodeUint(val.Bytes())), true
		}
	}
	if id == zed.IDDuration {
		return zed.DecodeInt(val.Bytes()), true
	}
	if id == zed.IDTime {
		return int64(zed.DecodeTime(val.Bytes())), true
	}
	if id == zed.IDString {
		v, err := strconv.ParseInt(string(val.Bytes()), 10, 64)
		return v, err == nil
	}
	return 0, false
}

func ToBool(val *zed.Value) (bool, bool) {
	if val.IsString() {
		v, err := strconv.ParseBool(string(val.Bytes()))
		return v, err == nil
	}
	v, ok := ToInt(val)
	return v != 0, ok
}

func ToTime(val *zed.Value) (nano.Ts, bool) {
	id := val.Type.ID()
	if id == zed.IDTime {
		return zed.DecodeTime(val.Bytes()), true
	}
	if zed.IsSigned(id) {
		return nano.Ts(zed.DecodeInt(val.Bytes())), true
	}
	if zed.IsInteger(id) {
		v := zed.DecodeUint(val.Bytes())
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		return nano.Ts(v), true
	}
	if zed.IsFloat(id) {
		return nano.Ts(zed.DecodeFloat(val.Bytes())), true
	}
	return 0, false
}

// ToDuration attempts to convert a value to a duration.  Int
// and Double are converted as seconds. The resulting coerced value is
// written to out, and true is returned. If the value cannot be
// coerced, then false is returned.
func ToDuration(in *zed.Value) (nano.Duration, bool) {
	switch in.Type.ID() {
	case zed.IDDuration:
		return zed.DecodeDuration(in.Bytes()), true
	case zed.IDUint16, zed.IDUint32, zed.IDUint64:
		v := zed.DecodeUint(in.Bytes())
		// check for overflow
		if v > math.MaxInt64 {
			return 0, false
		}
		return nano.Duration(v) * nano.Second, true
	case zed.IDInt16, zed.IDInt32, zed.IDInt64:
		v := zed.DecodeInt(in.Bytes())
		//XXX check for overflow here
		return nano.Duration(v) * nano.Second, true
	case zed.IDFloat16, zed.IDFloat32, zed.IDFloat64:
		return nano.Duration(zed.DecodeFloat(in.Bytes())), true
	}
	return 0, false
}
