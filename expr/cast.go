package expr

import (
	"math"
	"net"
	"unicode/utf8"

	"github.com/araddon/dateparse"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/nano"
)

type PrimitiveCaster func(zv zed.Value) (zed.Value, error)

func LookupPrimitiveCaster(typ zed.Type) PrimitiveCaster {
	switch typ {
	case zed.TypeBool:
		return castToBool
	case zed.TypeInt8:
		return castToInt8
	case zed.TypeInt16:
		return castToInt16
	case zed.TypeInt32:
		return castToInt32
	case zed.TypeInt64:
		return castToInt64
	case zed.TypeUint8:
		return castToUint8
	case zed.TypeUint16:
		return castToUint16
	case zed.TypeUint32:
		return castToUint32
	case zed.TypeUint64:
		return castToUint64
	case zed.TypeFloat32:
		return castToFloat32
	case zed.TypeFloat64:
		return castToFloat64
	case zed.TypeIP:
		return castToIP
	case zed.TypeNet:
		return castToNet
	case zed.TypeDuration:
		return castToDuration
	case zed.TypeTime:
		return castToTime
	case zed.TypeString:
		return castToString
	case zed.TypeBstring:
		return castToBstring
	case zed.TypeBytes:
		return castToBytes
	default:
		return nil
	}
}

var castToInt8 = castToIntN(zed.TypeInt8, math.MinInt8, math.MaxInt8)
var castToInt16 = castToIntN(zed.TypeInt16, math.MinInt16, math.MaxInt16)
var castToInt32 = castToIntN(zed.TypeInt32, math.MinInt32, math.MaxInt32)
var castToInt64 = castToIntN(zed.TypeInt64, 0, 0)

func castToIntN(typ zed.Type, min, max int64) func(zed.Value) (zed.Value, error) {
	return func(zv zed.Value) (zed.Value, error) {
		v, ok := coerce.ToInt(zv)
		// XXX better error message
		if !ok || (min != 0 && (v < min || v > max)) {
			return zed.Value{}, ErrBadCast
		}
		// XXX GC
		return zed.Value{typ, zed.EncodeInt(v)}, nil
	}
}

var castToUint8 = castToUintN(zed.TypeUint8, math.MaxUint8)
var castToUint16 = castToUintN(zed.TypeUint16, math.MaxUint16)
var castToUint32 = castToUintN(zed.TypeUint32, math.MaxUint32)
var castToUint64 = castToUintN(zed.TypeUint64, 0)

func castToUintN(typ zed.Type, max uint64) func(zed.Value) (zed.Value, error) {
	return func(zv zed.Value) (zed.Value, error) {
		v, ok := coerce.ToUint(zv)
		// XXX better error message
		if !ok || (max != 0 && v > max) {
			return zed.Value{}, ErrBadCast
		}
		// XXX GC
		return zed.Value{typ, zed.EncodeUint(v)}, nil
	}
}

func castToBool(zv zed.Value) (zed.Value, error) {
	b, ok := coerce.ToBool(zv)
	if !ok {
		return zed.Value{}, ErrBadCast
	}
	return zed.Value{zed.TypeBool, zed.EncodeBool(b)}, nil
}

func castToFloat32(zv zed.Value) (zed.Value, error) {
	f, ok := coerce.ToFloat(zv)
	if !ok {
		return zed.Value{}, ErrBadCast
	}
	return zed.Value{zed.TypeFloat32, zed.EncodeFloat32(float32(f))}, nil
}

func castToFloat64(zv zed.Value) (zed.Value, error) {
	f, ok := coerce.ToFloat(zv)
	if !ok {
		return zed.Value{}, ErrBadCast
	}
	return zed.Value{zed.TypeFloat64, zed.EncodeFloat64(f)}, nil
}

func castToIP(zv zed.Value) (zed.Value, error) {
	if !zv.IsStringy() {
		return zed.Value{}, ErrBadCast
	}
	ip := net.ParseIP(string(zv.Bytes))
	if ip == nil {
		return zed.Value{}, ErrBadCast
	}
	// XXX GC
	return zed.Value{zed.TypeIP, zed.EncodeIP(ip)}, nil
}

func castToNet(zv zed.Value) (zed.Value, error) {
	if !zv.IsStringy() {
		return zed.Value{}, ErrBadCast
	}
	_, net, err := net.ParseCIDR(string(zv.Bytes))
	if err != nil {
		return zed.Value{}, ErrBadCast
	}
	// XXX GC
	return zed.Value{zed.TypeNet, zed.EncodeNet(net)}, nil
}

func castToDuration(zv zed.Value) (zed.Value, error) {
	id := zv.Type.ID()
	if zed.IsStringy(id) {
		d, err := nano.ParseDuration(byteconv.UnsafeString(zv.Bytes))
		if err != nil {
			f, ferr := byteconv.ParseFloat64(zv.Bytes)
			if ferr != nil {
				return zed.NewError(err), nil
			}
			d = nano.DurationFromFloat(f)
		}
		// XXX GC
		return zed.Value{zed.TypeDuration, zed.EncodeDuration(d)}, nil
	}
	if zed.IsFloat(id) {
		f, err := zed.DecodeFloat(zv.Bytes)
		if err != nil {
			return zed.Value{}, err
		}
		d := nano.DurationFromFloat(f)
		// XXX GC
		return zed.Value{zed.TypeDuration, zed.EncodeDuration(d)}, nil
	}
	sec, ok := coerce.ToInt(zv)
	if !ok {
		return zed.Value{}, ErrBadCast
	}
	d := nano.Duration(sec) * nano.Second
	// XXX GC
	return zed.Value{zed.TypeDuration, zed.EncodeDuration(d)}, nil
}

func castToTime(zv zed.Value) (zed.Value, error) {
	id := zv.Type.ID()
	var ts nano.Ts
	switch {
	case zv.Bytes == nil:
		// Do nothing. Any nil value is cast to a zero time.
	case zed.IsStringy(id):
		gotime, err := dateparse.ParseAny(byteconv.UnsafeString(zv.Bytes))
		if err != nil {
			sec, ferr := byteconv.ParseFloat64(zv.Bytes)
			if ferr != nil {
				return zed.NewError(err), nil
			}
			ts = nano.Ts(1e9 * sec)
		} else {
			ts = nano.Ts(gotime.UnixNano())
		}
	case zed.IsFloat(id):
		sec, err := zed.DecodeFloat(zv.Bytes)
		if err != nil {
			return zed.Value{}, err
		}
		ts = nano.Ts(sec * 1e9)
	case zed.IsInteger(id):
		sec, ok := coerce.ToInt(zv)
		if !ok {
			return zed.NewErrorf("cannot convert value of type %s to time", zv.Type), nil
		}
		ts = nano.Ts(sec * 1e9)
	default:
		return zed.NewErrorf("cannot convert value of type %s to time", zv.Type), nil
	}
	return zed.Value{zed.TypeTime, zed.EncodeTime(ts)}, nil
}

func castToStringy(typ zed.Type) func(zed.Value) (zed.Value, error) {
	return func(zv zed.Value) (zed.Value, error) {
		id := zv.Type.ID()
		if id == zed.IDBytes || id == zed.IDBstring {
			if !utf8.Valid(zv.Bytes) {
				return zed.NewErrorf("non-UTF-8 bytes cannot be cast to string"), nil
			}
			return zed.Value{typ, zv.Bytes}, nil
		}
		if enum, ok := zv.Type.(*zed.TypeEnum); ok {
			selector, _ := zed.DecodeUint(zv.Bytes)
			symbol, err := enum.Symbol(int(selector))
			if err != nil {
				return zed.NewError(err), nil
			}
			return zed.Value{typ, zed.EncodeString(symbol)}, nil
		}
		if zed.IsStringy(id) {
			// If it's already stringy, then the Zeed encoding can stay
			// the same and we just update the stringy type.
			return zed.Value{typ, zv.Bytes}, nil
		}
		// Otherwise, we'll use a canonical ZSON value for the string rep
		// of an arbitrary value cast to a string.
		result := zv.Type.Format(zv.Bytes)
		return zed.Value{typ, zed.EncodeString(result)}, nil
	}
}

var castToString = castToStringy(zed.TypeString)
var castToBstring = castToStringy(zed.TypeBstring)

func castToBytes(zv zed.Value) (zed.Value, error) {
	return zed.Value{zed.TypeBytes, zed.EncodeBytes(zv.Bytes)}, nil
}
