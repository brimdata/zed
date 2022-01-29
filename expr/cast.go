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

func LookupPrimitiveCaster(zctx *zed.Context, typ zed.Type) Evaluator {
	switch typ {
	case zed.TypeBool:
		return &casterBool{zctx}
	case zed.TypeInt8:
		return &casterIntN{zctx, zed.TypeInt8, math.MinInt8, math.MaxInt8}
	case zed.TypeInt16:
		return &casterIntN{zctx, zed.TypeInt16, math.MinInt16, math.MaxInt16}
	case zed.TypeInt32:
		return &casterIntN{zctx, zed.TypeInt32, math.MinInt32, math.MaxInt32}
	case zed.TypeInt64:
		return &casterIntN{zctx, zed.TypeInt64, 0, 0}
	case zed.TypeUint8:
		return &casterUintN{zctx, zed.TypeUint8, math.MaxUint8}
	case zed.TypeUint16:
		return &casterUintN{zctx, zed.TypeUint16, math.MaxUint16}
	case zed.TypeUint32:
		return &casterUintN{zctx, zed.TypeUint32, math.MaxUint32}
	case zed.TypeUint64:
		return &casterUintN{zctx, zed.TypeUint64, 0}
	case zed.TypeFloat32:
		return &casterFloat32{zctx}
	case zed.TypeFloat64:
		return &casterFloat64{zctx}
	case zed.TypeIP:
		return &casterIP{zctx}
	case zed.TypeNet:
		return &casterNet{zctx}
	case zed.TypeDuration:
		return &casterDuration{zctx}
	case zed.TypeTime:
		return &casterTime{zctx}
	case zed.TypeString:
		return &casterString{zctx}
	case zed.TypeBytes:
		return &casterBytes{}
	default:
		return nil
	}
}

type casterIntN struct {
	zctx *zed.Context
	typ  zed.Type
	min  int64
	max  int64
}

func (c *casterIntN) Eval(ectx Context, val *zed.Value) *zed.Value {
	v, ok := coerce.ToInt(*val)
	if !ok || (c.min != 0 && (v < c.min || v > c.max)) {
		return ectx.CopyValue(*c.zctx.NewErrorf(
			"cannot cast %s to type %s", zson.MustFormatValue(*val), zson.FormatType(c.typ)))
	}
	return ectx.NewValue(c.typ, zed.EncodeInt(v))
}

type casterUintN struct {
	zctx *zed.Context
	typ  zed.Type
	max  uint64
}

func (c *casterUintN) Eval(ectx Context, val *zed.Value) *zed.Value {
	if val.Type == zed.TypeTime {
		return ectx.NewValue(c.typ, zed.EncodeUint(uint64(zed.DecodeTime(val.Bytes))))
	}
	v, ok := coerce.ToUint(*val)
	if !ok || (c.max != 0 && v > c.max) {
		return ectx.CopyValue(*c.zctx.NewErrorf(
			"cannot cast %s to type %s", zson.MustFormatValue(*val), zson.FormatType(c.typ)))
	}
	return ectx.NewValue(c.typ, zed.EncodeUint(v))
}

type casterBool struct {
	zctx *zed.Context
}

func (c *casterBool) Eval(ectx Context, val *zed.Value) *zed.Value {
	b, ok := coerce.ToBool(*val)
	if !ok {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to bool", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeBool, zed.EncodeBool(b))
}

type casterFloat32 struct {
	zctx *zed.Context
}

func (c *casterFloat32) Eval(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(*val)
	if !ok {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type float32", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeFloat32, zed.EncodeFloat32(float32(f)))
}

type casterFloat64 struct {
	zctx *zed.Context
}

func (c *casterFloat64) Eval(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(*val)
	if !ok {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type float64", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeFloat64, zed.EncodeFloat64(f))
}

type casterIP struct {
	zctx *zed.Context
}

func (c *casterIP) Eval(ectx Context, val *zed.Value) *zed.Value {
	if _, ok := zed.TypeUnder(val.Type).(*zed.TypeOfIP); ok {
		return val
	}
	if !val.IsString() {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type ip", zson.MustFormatValue(*val)))
	}
	ip, err := byteconv.ParseIP(val.Bytes)
	if err != nil {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type ip", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeIP, zed.EncodeIP(ip))
}

type casterNet struct {
	zctx *zed.Context
}

func (c *casterNet) Eval(ectx Context, val *zed.Value) *zed.Value {
	if !val.IsString() {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type net", zson.MustFormatValue(*val)))
	}
	// XXX GC
	_, net, err := net.ParseCIDR(string(val.Bytes))
	if err != nil {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type net", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeNet, zed.EncodeNet(net))
}

type casterDuration struct {
	zctx *zed.Context
}

func (c *casterDuration) Eval(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	if id == zed.IDString {
		d, err := nano.ParseDuration(byteconv.UnsafeString(val.Bytes))
		if err != nil {
			f, ferr := byteconv.ParseFloat64(val.Bytes)
			if ferr != nil {
				return ectx.CopyValue(*c.zctx.NewErrorf(
					"cannot cast %s to type duration", zson.MustFormatValue(*val)))
			}
			d = nano.DurationFromFloat(f)
		}
		return ectx.NewValue(zed.TypeDuration, zed.EncodeDuration(d))
	}
	if zed.IsFloat(id) {
		d := nano.DurationFromFloat(zed.DecodeFloat(val.Bytes))
		return ectx.NewValue(zed.TypeDuration, zed.EncodeDuration(d))
	}
	v, ok := coerce.ToInt(*val)
	if !ok {
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type duration", zson.MustFormatValue(*val)))
	}
	d := nano.Duration(v)
	return ectx.NewValue(zed.TypeDuration, zed.EncodeDuration(d))
}

type casterTime struct {
	zctx *zed.Context
}

func (c *casterTime) Eval(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	var ts nano.Ts
	switch {
	case val.Bytes == nil:
		// Do nothing. Any nil value is cast to a zero time.
		//XXX maybe this should be a null time not a zero time.
		return ectx.NewValue(zed.TypeTime, nil)
	case id == zed.IDTime:
		return val
	case id == zed.IDString:
		gotime, err := dateparse.ParseAny(byteconv.UnsafeString(val.Bytes))
		if err != nil {
			sec, ferr := byteconv.ParseFloat64(val.Bytes)
			if ferr != nil {
				return ectx.CopyValue(*c.zctx.NewErrorf(
					"cannot cast %s to type time", zson.MustFormatValue(*val)))
			}
			ts = nano.Ts(1e9 * sec)
		} else {
			ts = nano.Ts(gotime.UnixNano())
		}
	case zed.IsFloat(id):
		ts = nano.Ts(zed.DecodeFloat(val.Bytes) * 1e9)
	case zed.IsInteger(id):
		//XXX we call coerce here to avoid unsigned/signed decode
		v, ok := coerce.ToInt(*val)
		if !ok {
			panic("coerce int to int failed")
		}
		ts = nano.Ts(v)
	default:
		return ectx.CopyValue(*c.zctx.NewErrorf("cannot cast %s to type time", zson.MustFormatValue(*val)))
	}
	return ectx.NewValue(zed.TypeTime, zed.EncodeTime(ts))
}

type casterString struct {
	zctx *zed.Context
}

func (c *casterString) Eval(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	if id == zed.IDBytes {
		if !utf8.Valid(val.Bytes) {
			return ectx.CopyValue(*c.zctx.NewErrorf("non-UTF-8 bytes cannot be cast to type string"))
		}
		return ectx.NewValue(zed.TypeString, val.Bytes)
	}
	if enum, ok := val.Type.(*zed.TypeEnum); ok {
		selector := zed.DecodeUint(val.Bytes)
		symbol, err := enum.Symbol(int(selector))
		if err != nil {
			return ectx.CopyValue(*c.zctx.NewError(err))
		}
		return ectx.NewValue(zed.TypeString, zed.EncodeString(symbol))
	}
	if id == zed.IDString {
		// If it's already stringy, then the Zed encoding can stay
		// the same and we just update the stringy type.
		return ectx.NewValue(zed.TypeString, val.Bytes)
	}
	// Otherwise, we'll use a canonical ZSON value for the string rep
	// of an arbitrary value cast to a string.
	result := zson.MustFormatValue(*val)
	return ectx.NewValue(zed.TypeString, zed.EncodeString(result))
}

type casterBytes struct {
	zctx *zed.Context
}

func (c *casterBytes) Eval(ectx Context, val *zed.Value) *zed.Value {
	return ectx.NewValue(zed.TypeBytes, val.Bytes)
}
