package expr

import (
	"errors"
	"fmt"
	"math"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zngnative"
)

type Function func([]zngnative.NativeValue) (zngnative.NativeValue, error)

var ErrTooFewArgs = errors.New("too few arguments")
var ErrTooManyArgs = errors.New("too many arguments")
var ErrBadArgument = errors.New("bad argument")

var allFns = map[string]struct {
	minArgs int
	maxArgs int
	impl    Function
}{
	"Math.max":  {1, -1, mathMax},
	"Math.min":  {1, -1, mathMin},
	"Math.sqrt": {1, 1, mathSqrt},
}

func mathMax(args []zngnative.NativeValue) (zngnative.NativeValue, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		ret := args[0].Value.(int64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToInt(val)
			if ok && v > ret {
				ret = v
			}
		}
		return zngnative.NativeValue{zng.TypeInt64, ret}, nil

	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		ret := args[0].Value.(uint64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToUint(val)
			if ok && v > ret {
				ret = v
			}
		}
		return zngnative.NativeValue{zng.TypeUint64, ret}, nil

	case zng.IdFloat64:
		ret := args[0].Value.(float64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToFloat64(val)
			if ok && v > ret {
				ret = v
			}
		}
		return zngnative.NativeValue{zng.TypeFloat64, ret}, nil

	default:
		return zngnative.NativeValue{}, fmt.Errorf("Math.max: %w", ErrBadArgument)
	}
}

func mathMin(args []zngnative.NativeValue) (zngnative.NativeValue, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		ret := args[0].Value.(int64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToInt(val)
			if ok && v < ret {
				ret = v
			}
		}
		return zngnative.NativeValue{zng.TypeInt64, ret}, nil

	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		ret := args[0].Value.(uint64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToUint(val)
			if ok && v < ret {
				ret = v
			}
		}
		return zngnative.NativeValue{zng.TypeUint64, ret}, nil

	case zng.IdFloat64:
		ret := args[0].Value.(float64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToFloat64(val)
			if ok && v < ret {
				ret = v
			}
		}
		return zngnative.NativeValue{zng.TypeFloat64, ret}, nil

	default:
		return zngnative.NativeValue{}, fmt.Errorf("Math.min: %w", ErrBadArgument)
	}
}

func mathSqrt(args []zngnative.NativeValue) (zngnative.NativeValue, error) {
	var x float64
	switch args[0].Type.ID() {
	case zng.IdFloat64:
		x = args[0].Value.(float64)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		x = float64(args[0].Value.(int64))
	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		x = float64(args[0].Value.(uint64))
	default:
		return zngnative.NativeValue{}, fmt.Errorf("Math.sqrt: %w", ErrBadArgument)
	}

	r := math.Sqrt(x)
	if math.IsNaN(r) {
		// For now we can't represent non-numeric values in a float64,
		// we will revisit this but it has implications for file
		// formats, zql, etc.
		return zngnative.NativeValue{}, fmt.Errorf("Math.sqrt: %w", ErrBadArgument)
	}

	return zngnative.NativeValue{zng.TypeFloat64, r}, nil
}
