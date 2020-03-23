package expr

import (
	"errors"
	"fmt"
	"math"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zngnative"
)

type Function func([]zngnative.NativeValue) (zngnative.NativeValue, error)

var ErrWrongArgc = errors.New("wrong number of arguments")
var ErrBadArgument = errors.New("bad argument")

var allFns = map[string]Function{
	"Math.sqrt": mathSqrt,
}

func mathSqrt(args []zngnative.NativeValue) (zngnative.NativeValue, error) {
	if len(args) < 1 || len(args) > 1 {
		return zngnative.NativeValue{}, fmt.Errorf("Math.sqrt: %w", ErrWrongArgc)
	}

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
