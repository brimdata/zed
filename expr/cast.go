package expr

import (
	"math"
	"net"

	"github.com/brimsec/zq/expr/coerce"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

type ValueCaster func(zv zng.Value) (zng.Value, error)

func LookupValueCaster(typ string) ValueCaster {
	switch typ {
	case "int8":
		return castToInt8
	case "int16":
		return castToInt16
	case "int32":
		return castToInt32
	case "int64":
		return castToInt64
	case "uint8":
		return castToUint8
	case "uint16":
		return castToUint16
	case "uint32":
		return castToUint32
	case "uint64":
		return castToUint64
	case "float64":
		return castToFloat64
	case "ip":
		return castToIP
	case "time":
		return castToTime
	case "string":
		return castToString
	case "bstring":
		return castToBstring
	case "bytes":
		return castToBytes
	default:
		return nil
	}
}

func castToIntN(min, max int64, typ zng.Type) func(zng.Value) (zng.Value, error) {
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

var castToInt8 = castToIntN(math.MinInt8, math.MaxInt8, zng.TypeInt8)
var castToInt16 = castToIntN(math.MinInt16, math.MaxInt16, zng.TypeInt16)
var castToInt32 = castToIntN(math.MinInt32, math.MaxInt32, zng.TypeInt32)
var castToInt64 = castToIntN(0, 0, zng.TypeInt64)

func castToUintN(max uint64, typ zng.Type) func(zng.Value) (zng.Value, error) {
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

var castToUint8 = castToUintN(math.MaxUint8, zng.TypeUint8)
var castToUint16 = castToUintN(math.MaxUint16, zng.TypeUint16)
var castToUint32 = castToUintN(math.MaxUint32, zng.TypeUint32)
var castToUint64 = castToUintN(0, zng.TypeUint64)

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

func castToTime(zv zng.Value) (zng.Value, error) {
	if zng.IsFloat(zv.Type.ID()) {
		f, _ := zng.DecodeFloat64(zv.Bytes)
		ts := nano.FloatToTs(f)
		// XXX GC
		return zng.Value{zng.TypeTime, zng.EncodeTime(ts)}, nil
	}
	ns, ok := coerce.ToInt(zv)
	if !ok {
		return zng.Value{}, ErrBadCast
	}
	return zng.Value{zng.TypeTime, zng.EncodeTime(nano.Ts(ns))}, nil
}

func castToStringy(typ zng.Type) func(zng.Value) (zng.Value, error) {
	return func(zv zng.Value) (zng.Value, error) {
		if zv.Type.ID() == zng.IdBytes {
			return zng.Value{typ, zng.EncodeString(string(zv.Bytes))}, nil
		}
		if enum, ok := zv.Type.(*zng.TypeEnum); ok {
			selector, _ := zng.DecodeUint(zv.Bytes)
			element, err := enum.Element(int(selector))
			if err != nil {
				return zng.NewError(err), nil
			}
			return zng.Value{typ, zng.EncodeString(element.Name)}, nil
		}
		//XXX here, we need to create a human-readable string rep
		// rather than a tzng encoding, e.g., for time, an iso date instead of
		// ns int.  For now, this works for numbers and IPs.  We will fix in a
		// subsequent PR (see issue #1603).
		result := zv.Type.StringOf(zv.Bytes, zng.OutFormatUnescaped, false)
		return zng.Value{typ, zng.EncodeString(result)}, nil
	}
}

var castToString = castToStringy(zng.TypeString)
var castToBstring = castToStringy(zng.TypeBstring)

func castToBytes(zv zng.Value) (zng.Value, error) {
	return zng.Value{zng.TypeBytes, zng.EncodeBytes(zv.Bytes)}, nil
}
