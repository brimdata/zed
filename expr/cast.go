package expr

import (
	"math"
	"net"
	"unicode/utf8"

	"github.com/araddon/dateparse"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zson"
)

type Caster func(zv *zed.Value) *zed.Value

func LookupPrimitiveCaster(typ zed.Type) Caster {
	switch typ {
	case zed.TypeBool:
		return newBoolCaster()
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
		return newFloat32Caster()
	case zed.TypeFloat64:
		return newFloat64Caster()
	case zed.TypeIP:
		return newIPCaster()
	case zed.TypeNet:
		return newNetCaster()
	case zed.TypeDuration:
		return newDurationCaster()
	case zed.TypeTime:
		return newTimeCaster()
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

func castToIntN(typ zed.Type, min, max int64) Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		v, ok := coerce.ToInt(*zv)
		if !ok || (min != 0 && (v < min || v > max)) {
			return stash.Errorf("cannot cast %s to unsigned integer", zson.FormatType(zv.Type))
		}
		return stash.CopyVal(zed.Value{typ, zed.EncodeInt(v)})
	}
}

var castToUint8 = castToUintN(zed.TypeUint8, math.MaxUint8)
var castToUint16 = castToUintN(zed.TypeUint16, math.MaxUint16)
var castToUint32 = castToUintN(zed.TypeUint32, math.MaxUint32)
var castToUint64 = castToUintN(zed.TypeUint64, 0)

func castToUintN(typ zed.Type, max uint64) Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		v, ok := coerce.ToUint(*zv)
		if !ok || (max != 0 && v > max) {
			return stash.Errorf("cannot cast %s to unsigned integer", zson.FormatType(zv.Type))
		}
		return stash.CopyVal(zed.Value{typ, zed.EncodeUint(v)})
	}
}

//XXX remove ErrBadCast and other unused Errs

func newBoolCaster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		b, ok := coerce.ToBool(*zv)
		if !ok {
			return stash.Errorf("cannot cast %s to bool", zson.FormatType(zv.Type))
		}
		return stash.CopyVal(zed.Value{zed.TypeBool, zed.EncodeBool(b)})
	}
}

func newFloat32Caster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		f, ok := coerce.ToFloat(*zv)
		if !ok {
			return stash.Errorf("cannot cast %s to float32", zson.FormatType(zv.Type))
		}
		return stash.CopyVal(zed.Value{zed.TypeFloat32, zed.EncodeFloat32(float32(f))})
	}
}

func newFloat64Caster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		f, ok := coerce.ToFloat(*zv)
		if !ok {
			return stash.Errorf("cannot cast %s to float64", zson.FormatType(zv.Type))
		}
		return stash.CopyVal(zed.Value{zed.TypeFloat64, zed.EncodeFloat64(f)})
	}
}

func newIPCaster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		//XXX move same type check above with the null check?
		if _, ok := zed.AliasOf(zv.Type).(*zed.TypeOfIP); ok {
			return zv
		}
		if !zv.IsStringy() {
			return stash.Errorf("cannot cast %s to ip", zson.FormatType(zv.Type))
		}
		// XXX GC
		ip := net.ParseIP(string(zv.Bytes))
		if ip == nil {
			return stash.Errorf("cannot cast to ip: %q", string(zv.Bytes))
		}
		return stash.CopyVal(zed.Value{zed.TypeIP, zed.EncodeIP(ip)})
	}
}

func newNetCaster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		if !zv.IsStringy() {
			return stash.Errorf("cannot cast %s to net", zson.FormatType(zv.Type))
		}
		// XXX GC
		_, net, err := net.ParseCIDR(string(zv.Bytes))
		if err != nil {
			return stash.Errorf("cannot cast to net: %q", string(zv.Bytes))
		}
		return stash.CopyVal(zed.Value{zed.TypeNet, zed.EncodeNet(net)})
	}
}

func newDurationCaster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		id := zv.Type.ID()
		if zed.IsStringy(id) {
			d, err := nano.ParseDuration(byteconv.UnsafeString(zv.Bytes))
			if err != nil {
				f, ferr := byteconv.ParseFloat64(zv.Bytes)
				if ferr != nil {
					return stash.Errorf("cannot cast %q to duration", string(zv.Bytes))
				}
				d = nano.DurationFromFloat(f)
			}
			return stash.CopyVal(zed.Value{zed.TypeDuration, zed.EncodeDuration(d)})
		}
		if zed.IsFloat(id) {
			f, err := zed.DecodeFloat(zv.Bytes)
			if err != nil {
				panic(err)
			}
			d := nano.DurationFromFloat(f)
			return stash.CopyVal(zed.Value{zed.TypeDuration, zed.EncodeDuration(d)})
		}
		sec, ok := coerce.ToInt(*zv)
		if !ok {
			return stash.Errorf("cannot cast type %s to duration", zson.FormatType(zv.Type))
		}
		d := nano.Duration(sec) * nano.Second
		return stash.CopyVal(zed.Value{zed.TypeDuration, zed.EncodeDuration(d)})
	}
}

func newTimeCaster() Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
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
					return stash.Errorf("cannot cast %q to time", string(zv.Bytes))
				}
				ts = nano.Ts(1e9 * sec)
			} else {
				ts = nano.Ts(gotime.UnixNano())
			}
		case zed.IsFloat(id):
			sec, err := zed.DecodeFloat(zv.Bytes)
			if err != nil {
				panic(err)
			}
			ts = nano.Ts(sec * 1e9)
		case zed.IsInteger(id):
			//XXX we call coerce here to avoid unsigned/signed decode
			sec, ok := coerce.ToInt(*zv)
			if !ok {
				panic("coerce int to int failed")
			}
			ts = nano.Ts(sec * 1e9)
		default:
			return stash.Errorf("cannot cast type %s to time", zson.FormatType(zv.Type))
		}
		return stash.CopyVal(zed.Value{zed.TypeTime, zed.EncodeTime(ts)})
	}
}

//

func newStringyCaster(typ zed.Type) Caster {
	var stash result.Value
	return func(zv *zed.Value) *zed.Value {
		id := zv.Type.ID()
		if id == zed.IDBytes || id == zed.IDBstring {
			if !utf8.Valid(zv.Bytes) {
				return stash.Errorf("non-UTF-8 bytes cannot be cast to string")
			}
			return stash.CopyVal(zed.Value{typ, zv.Bytes})
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
