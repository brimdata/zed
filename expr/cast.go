package expr

import (
	"math"
	"net"
	"time"
	"unicode/utf8"

	"github.com/brimsec/zq/expr/coerce"
	"github.com/brimsec/zq/expr/function"
	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

type PrimitiveCaster func(zv zng.Value) (zng.Value, error)

func LookupPrimitiveCaster(typ zng.Type) PrimitiveCaster {
	switch typ {
	case zng.TypeInt8:
		return castToInt8
	case zng.TypeInt16:
		return castToInt16
	case zng.TypeInt32:
		return castToInt32
	case zng.TypeInt64:
		return castToInt64
	case zng.TypeUint8:
		return castToUint8
	case zng.TypeUint16:
		return castToUint16
	case zng.TypeUint32:
		return castToUint32
	case zng.TypeUint64:
		return castToUint64
	case zng.TypeFloat64:
		return castToFloat64
	case zng.TypeIP:
		return castToIP
	case zng.TypeDuration:
		return castToDuration
	case zng.TypeTime:
		return castToTime
	case zng.TypeString:
		return castToString
	case zng.TypeBstring:
		return castToBstring
	case zng.TypeBytes:
		return castToBytes
	default:
		return nil
	}
}

var castToInt8 = castToIntN(zng.TypeInt8, math.MinInt8, math.MaxInt8)
var castToInt16 = castToIntN(zng.TypeInt16, math.MinInt16, math.MaxInt16)
var castToInt32 = castToIntN(zng.TypeInt32, math.MinInt32, math.MaxInt32)
var castToInt64 = castToIntN(zng.TypeInt64, 0, 0)

func castToIntN(typ zng.Type, min, max int64) func(zng.Value) (zng.Value, error) {
	return func(zv zng.Value) (zng.Value, error) {
		v, ok := coerce.ToInt(zv)
		// XXX better error message
		if !ok || (min != 0 && (v < min || v > max)) {
			return zng.Value{}, ErrBadCast
		}
		// XXX GC
		return zng.Value{typ, zng.EncodeInt(v)}, nil
	}
}

var castToUint8 = castToUintN(zng.TypeUint8, math.MaxUint8)
var castToUint16 = castToUintN(zng.TypeUint16, math.MaxUint16)
var castToUint32 = castToUintN(zng.TypeUint32, math.MaxUint32)
var castToUint64 = castToUintN(zng.TypeUint64, 0)

func castToUintN(typ zng.Type, max uint64) func(zng.Value) (zng.Value, error) {
	return func(zv zng.Value) (zng.Value, error) {
		v, ok := coerce.ToUint(zv)
		// XXX better error message
		if !ok || (max != 0 && v > max) {
			return zng.Value{}, ErrBadCast
		}
		// XXX GC
		return zng.Value{typ, zng.EncodeUint(v)}, nil
	}
}

func castToFloat64(zv zng.Value) (zng.Value, error) {
	f, ok := coerce.ToFloat(zv)
	if !ok {
		return zng.Value{}, ErrBadCast
	}
	return zng.Value{zng.TypeFloat64, zng.EncodeFloat64(f)}, nil
}

func castToIP(zv zng.Value) (zng.Value, error) {
	if !zv.IsStringy() {
		return zng.Value{}, ErrBadCast
	}
	ip := net.ParseIP(string(zv.Bytes))
	if ip == nil {
		return zng.Value{}, ErrBadCast
	}
	// XXX GC
	return zng.Value{zng.TypeIP, zng.EncodeIP(ip)}, nil
}

func castToDuration(zv zng.Value) (zng.Value, error) {
	id := zv.Type.ID()
	if zng.IsStringy(id) {
		d, err := time.ParseDuration(byteconv.UnsafeString(zv.Bytes))
		var ns int64
		if err == nil {
			ns = int64(d)
		} else {
			f, ferr := byteconv.ParseFloat64(zv.Bytes)
			if ferr != nil {
				return zng.NewError(err), nil
			}
			ns = int64(f * 1e9)
		}
		return zng.Value{zng.TypeDuration, zng.EncodeDuration(ns)}, nil
	}
	if zng.IsFloat(id) {
		f, _ := zng.DecodeFloat64(zv.Bytes)
		ts := int64(nano.FloatToTs(f))
		// XXX GC
		return zng.Value{zng.TypeDuration, zng.EncodeDuration(ts)}, nil
	}
	ns, ok := coerce.ToInt(zv)
	if !ok {
		return zng.Value{}, ErrBadCast
	}
	return zng.Value{zng.TypeDuration, zng.EncodeDuration(1_000_000_000 * ns)}, nil
}

func castToTime(zv zng.Value) (zng.Value, error) {
	ts, err := function.CastToTime(zv)
	if err != nil {
		return zng.NewError(err), nil
	}
	return zng.Value{zng.TypeTime, zng.EncodeTime(ts)}, nil
}

func castToStringy(typ zng.Type) func(zng.Value) (zng.Value, error) {
	return func(zv zng.Value) (zng.Value, error) {
		id := zv.Type.ID()
		if id == zng.IdBytes || id == zng.IdBstring {
			if !utf8.Valid(zv.Bytes) {
				return zng.NewErrorf("non-UTF-8 bytes cannot be cast to string"), nil
			}
			return zng.Value{typ, zv.Bytes}, nil
		}
		if enum, ok := zv.Type.(*zng.TypeEnum); ok {
			selector, _ := zng.DecodeUint(zv.Bytes)
			element, err := enum.Element(int(selector))
			if err != nil {
				return zng.NewError(err), nil
			}
			return zng.Value{typ, zng.EncodeString(element.Name)}, nil
		}
		if zng.IsStringy(id) {
			// If it's already stringy, then the z encoding can stay
			// the same and we just update the stringy type.
			return zng.Value{typ, zv.Bytes}, nil
		}
		// Otherwise, we'll use a canonical ZSON value for the string rep
		// of an arbitrary value cast to a string.
		result := zv.Type.ZSONOf(zv.Bytes)
		return zng.Value{typ, zng.EncodeString(result)}, nil
	}
}

var castToString = castToStringy(zng.TypeString)
var castToBstring = castToStringy(zng.TypeBstring)

func castToBytes(zv zng.Value) (zng.Value, error) {
	return zng.Value{zng.TypeBytes, zng.EncodeBytes(zv.Bytes)}, nil
}
