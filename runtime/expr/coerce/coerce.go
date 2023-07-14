package coerce

import (
	"bytes"
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/constraints"
)

func Equal(a, b *zed.Value) bool {
	if a.IsNull() {
		return b.IsNull()
	} else if b.IsNull() {
		// We know a isn't null.
		return false
	}
	switch aid, bid := a.Type.ID(), b.Type.ID(); {
	case !zed.IsNumber(aid) || !zed.IsNumber(bid):
		return aid == bid && bytes.Equal(a.Bytes(), b.Bytes())
	case zed.IsUnsigned(aid) && zed.IsUnsigned(bid):
		return a.Uint() == b.Uint()
	case zed.IsFloat(aid):
		return a.Float() == ToNumeric[float64](b)
	case zed.IsFloat(bid):
		return b.Float() == ToNumeric[float64](a)
	case zed.IsUnsigned(aid):
		v := a.Uint()
		return v <= math.MaxInt64 && int64(v) == b.Int()
	case zed.IsUnsigned(bid):
		v := b.Uint()
		return v <= math.MaxInt64 && int64(v) == a.Int()
	default:
		return a.Int() == b.Int()
	}
}

func ToNumeric[T constraints.Integer | constraints.Float](val *zed.Value) T {
	switch id := val.Type.ID(); {
	case zed.IsUnsigned(id):
		return T(val.Uint())
	case zed.IsSigned(id):
		return T(val.Int())
	case zed.IsFloat(id):
		return T(val.Float())
	}
	panic(zson.FormatValue(val))
}

func ToFloat(val *zed.Value) (float64, bool) {
	switch id := val.Type.ID(); {
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

func ToUint(val *zed.Value) (uint64, bool) {
	switch id := val.Type.ID(); {
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

func ToInt(val *zed.Value) (int64, bool) {
	switch id := val.Type.ID(); {
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

func ToBool(val *zed.Value) (bool, bool) {
	if val.IsString() {
		v, err := byteconv.ParseBool(val.Bytes())
		return v, err == nil
	}
	v, ok := ToInt(val)
	return v != 0, ok
}
