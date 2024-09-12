package coerce

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/constraints"
)

func Equal(a, b zed.Value) bool {
	if a.IsNull() {
		return b.IsNull()
	} else if b.IsNull() {
		// We know a isn't null.
		return false
	}
	switch aid, bid := a.Type().ID(), b.Type().ID(); {
	case !zed.IsNumber(aid) || !zed.IsNumber(bid):
		return aid == bid && bytes.Equal(a.Bytes(), b.Bytes())
	case zed.IsFloat(aid):
		return a.Float() == ToNumeric[float64](b)
	case zed.IsFloat(bid):
		return b.Float() == ToNumeric[float64](a)
	case zed.IsSigned(aid):
		av := a.Int()
		if zed.IsUnsigned(bid) {
			return uint64(av) == b.Uint() && av >= 0
		}
		return av == b.Int()
	case zed.IsSigned(bid):
		bv := b.Int()
		if zed.IsUnsigned(aid) {
			return uint64(bv) == a.Uint() && bv >= 0
		}
		return bv == a.Int()
	default:
		return a.Uint() == b.Uint()
	}
}

func ToNumeric[T constraints.Integer | constraints.Float](val zed.Value) T {
	if val.IsNull() {
		return 0
	}
	val = val.Under()
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id):
		return T(val.Uint())
	case zed.IsSigned(id):
		return T(val.Int())
	case zed.IsFloat(id):
		return T(val.Float())
	}
	panic(zson.FormatValue(val))
}

func ToFloat(val zed.Value) (float64, bool) {
	val = val.Under()
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id):
		return float64(val.Uint()), true
	case zed.IsSigned(id):
		return float64(val.Int()), true
	case zed.IsFloat(id):
		return val.Float(), true
	case id == zed.IDString:
		v, err := byteconv.ParseFloat64(val.Bytes())
		return v, err == nil
	}
	return 0, false
}

func ToUint(val zed.Value) (uint64, bool) {
	val = val.Under()
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id):
		return val.Uint(), true
	case zed.IsSigned(id):
		v := val.Int()
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case zed.IsFloat(id):
		return uint64(val.Float()), true
	case id == zed.IDString:
		v, err := byteconv.ParseUint64(val.Bytes())
		return v, err == nil
	}
	return 0, false
}

func ToInt(val zed.Value) (int64, bool) {
	val = val.Under()
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id):
		return int64(val.Uint()), true
	case zed.IsSigned(id):
		// XXX check if negative? should -1:uint64 be maxint64 or an error?
		return val.Int(), true
	case zed.IsFloat(id):
		return int64(val.Float()), true
	case id == zed.IDString:
		v, err := byteconv.ParseInt64(val.Bytes())
		return v, err == nil
	}
	return 0, false
}

func ToBool(val zed.Value) (bool, bool) {
	val = val.Under()
	if val.IsString() {
		v, err := byteconv.ParseBool(val.Bytes())
		return v, err == nil
	}
	v, ok := ToInt(val)
	return v != 0, ok
}
