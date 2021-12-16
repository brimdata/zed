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
	"github.com/brimdata/zed/zson"
)

type Caster func(Context, *zed.Value) *zed.Value

func LookupPrimitiveCaster(typ zed.Type) Caster {
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
	case zed.TypeError:
		return castToError
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

func castToIntN(typ zed.Type, min, max int64) Caster {
	return func(ectx Context, val *zed.Value) *zed.Value {
		v, ok := coerce.ToInt(*val)
		if !ok || (min != 0 && (v < min || v > max)) {
			return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type %s", zson.MustFormatValue(*val), zson.FormatType(typ)))
		}
		return ectx.NewValue(typ, zed.EncodeInt(v))
	}
}

var castToUint8 = castToUintN(zed.TypeUint8, math.MaxUint8)
var castToUint16 = castToUintN(zed.TypeUint16, math.MaxUint16)
var castToUint32 = castToUintN(zed.TypeUint32, math.MaxUint32)
var castToUint64 = castToUintN(zed.TypeUint64, 0)

func castToUintN(typ zed.Type, max uint64) Caster {
	return func(ectx Context, val *zed.Value) *zed.Value {
		v, ok := coerce.ToUint(*val)
		if !ok || (max != 0 && v > max) {
			return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type %s", zson.MustFormatValue(*val), zson.FormatType(typ)))
		}
		return ectx.NewValue(typ, zed.EncodeUint(v))
	}
}

func castToBool(ectx Context, val *zed.Value) *zed.Value {
	b, ok := coerce.ToBool(*val)
	if !ok {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to bool", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeBool, zed.EncodeBool(b))

}

func castToFloat32(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(*val)
	if !ok {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type float32", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeFloat32, zed.EncodeFloat32(float32(f)))
}

func castToFloat64(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(*val)
	if !ok {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type float64", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeFloat64, zed.EncodeFloat64(f))
}

func castToIP(ectx Context, val *zed.Value) *zed.Value {
	if _, ok := zed.AliasOf(val.Type).(*zed.TypeOfIP); ok {
		return val
	}
	if !val.IsStringy() {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type ip", zson.MustFormatValue(*val)))
	}
	// XXX GC
	ip := net.ParseIP(string(val.Bytes))
	if ip == nil {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type ip", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeIP, zed.EncodeIP(ip))
}

func castToNet(ectx Context, val *zed.Value) *zed.Value {
	if !val.IsStringy() {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type net", zson.MustFormatValue(*val)))
	}
	// XXX GC
	_, net, err := net.ParseCIDR(string(val.Bytes))
	if err != nil {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type net", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeNet, zed.EncodeNet(net))
}

func castToDuration(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	if zed.IsStringy(id) {
		d, err := nano.ParseDuration(byteconv.UnsafeString(val.Bytes))
		if err != nil {
			f, ferr := byteconv.ParseFloat64(val.Bytes)
			if ferr != nil {
				return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type duration", zson.MustFormatValue(*val)))
			}
			d = nano.DurationFromFloat(f)
		}
		return ectx.NewValue(zed.TypeDuration, zed.EncodeDuration(d))
	}
	if zed.IsFloat(id) {
		f, err := zed.DecodeFloat(val.Bytes)
		if err != nil {
			panic(err)
		}
		d := nano.DurationFromFloat(f)
		return ectx.NewValue(zed.TypeDuration, zed.EncodeDuration(d))
	}
	sec, ok := coerce.ToInt(*val)
	if !ok {
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type duration", zson.MustFormatValue(*val)))
	}
	d := nano.Duration(sec) * nano.Second
	return ectx.NewValue(zed.TypeDuration, zed.EncodeDuration(d))
}

func castToTime(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	var ts nano.Ts
	switch {
	case val.Bytes == nil:
		// Do nothing. Any nil value is cast to a zero time.
	case zed.IsStringy(id):
		gotime, err := dateparse.ParseAny(byteconv.UnsafeString(val.Bytes))
		if err != nil {
			sec, ferr := byteconv.ParseFloat64(val.Bytes)
			if ferr != nil {
				return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type time", zson.MustFormatValue(*val)))
			}
			ts = nano.Ts(1e9 * sec)
		} else {
			ts = nano.Ts(gotime.UnixNano())
		}
	case zed.IsFloat(id):
		sec, err := zed.DecodeFloat(val.Bytes)
		if err != nil {
			panic(err)
		}
		ts = nano.Ts(sec * 1e9)
	case zed.IsInteger(id):
		//XXX we call coerce here to avoid unsigned/signed decode
		sec, ok := coerce.ToInt(*val)
		if !ok {
			panic("coerce int to int failed")
		}
		ts = nano.Ts(sec * 1e9)
	default:
		return ectx.CopyValue(zed.NewErrorf("cannot cast %s to type time", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeTime, zed.EncodeTime(ts))
}

var castToString = castToStringy(zed.TypeString)
var castToBstring = castToStringy(zed.TypeBstring)
var castToError = castToStringy(zed.TypeError)

func castToStringy(typ zed.Type) Caster {
	return func(ectx Context, val *zed.Value) *zed.Value {
		id := val.Type.ID()
		if id == zed.IDBytes || id == zed.IDBstring {
			if !utf8.Valid(val.Bytes) {
				return ectx.CopyValue(zed.NewErrorf("non-UTF-8 bytes cannot be cast to type string"))
			}
			return ectx.NewValue(typ, val.Bytes)
		}
		if enum, ok := val.Type.(*zed.TypeEnum); ok {
			selector, _ := zed.DecodeUint(val.Bytes)
			symbol, err := enum.Symbol(int(selector))
			if err != nil {
				return ectx.CopyValue(zed.NewError(err))
			}
			return ectx.NewValue(typ, zed.EncodeString(symbol))
		}
		if zed.IsStringy(id) {
			// If it's already stringy, then the Zed encoding can stay
			// the same and we just update the stringy type.
			return ectx.NewValue(typ, val.Bytes)
		}
		// Otherwise, we'll use a canonical ZSON value for the string rep
		// of an arbitrary value cast to a string.
		result := zson.MustFormatValue(*val)
		return ectx.NewValue(typ, zed.EncodeString(result))
	}
}

func castToBytes(ectx Context, val *zed.Value) *zed.Value {
	return ectx.NewValue(zed.TypeBytes, val.Bytes)
}
