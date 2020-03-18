package expr

import (
	"errors"
	"fmt"
	"math"

	"github.com/brimsec/zq/zng"
)

type Function func([]NativeValue) (NativeValue, error)

var ErrWrongArgc = errors.New("wrong number of arguments")
var ErrBadArgument = errors.New("bad argument")

var allFns = map[string]Function{
	"Math.sqrt": mathSqrt,
}

func mathSqrt(args []NativeValue) (NativeValue, error) {
	if len(args) < 1 || len(args) > 1 {
		return NativeValue{}, fmt.Errorf("Math.sqrt: %w", ErrWrongArgc)
	}

	var x float64
	switch args[0].typ.ID() {
	case zng.IdFloat64:
		x = args[0].value.(float64)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		x = float64(args[0].value.(int64))
	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		x = float64(args[0].value.(uint64))
	default:
		return NativeValue{}, fmt.Errorf("Math.sqrt: %w", ErrBadArgument)
	}

	r := math.Sqrt(x)
	if math.IsNaN(r) {
		// For now we can't represent non-numeric values in a float64,
		// we will revisit this but it has implications for file
		// formats, zql, etc.
		return NativeValue{}, fmt.Errorf("Math.sqrt: %w", ErrBadArgument)
	}

	return NativeValue{zng.TypeFloat64, r}, nil
}
