package expr_test

import (
	"testing"

	"github.com/brimsec/zq/expr"
	"github.com/stretchr/testify/require"
)

func TestBadFunction(t *testing.T) {
	testError(t, "notafunction()", nil, expr.ErrNoSuchFunction, "calling nonexistent function")
}

func TestSqrt(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[f:float64,i:int32]
0:[6.25;9;]`)
	require.NoError(t, err)

	testSuccessful(t, "Math.sqrt(4.0)", record, zfloat64(2.0))
	testSuccessful(t, "Math.sqrt(f)", record, zfloat64(2.5))
	testSuccessful(t, "Math.sqrt(i)", record, zfloat64(3.0))

	testError(t, "Math.sqrt()", record, expr.ErrTooFewArgs, "sqrt with no args")
	testError(t, "Math.sqrt(1, 2)", record, expr.ErrTooManyArgs, "sqrt with too many args")
	testError(t, "Math.sqrt(-1)", record, expr.ErrBadArgument, "sqrt of negative")
}

func TestMinMax(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[i:uint64,f:float64]
0:[1;2;]`)
	require.NoError(t, err)

	// Simple cases
	testSuccessful(t, "Math.min(1)", record, zint64(1))
	testSuccessful(t, "Math.max(1)", record, zint64(1))
	testSuccessful(t, "Math.min(1, 2, 3)", record, zint64(1))
	testSuccessful(t, "Math.max(1, 2, 3)", record, zint64(3))
	testSuccessful(t, "Math.min(3, 2, 1)", record, zint64(1))
	testSuccessful(t, "Math.max(3, 2, 1)", record, zint64(3))

	// Fails with no arguments
	testError(t, "Math.min()", record, expr.ErrTooFewArgs, "min with no args")
	testError(t, "Math.max()", record, expr.ErrTooFewArgs, "max with no args")

	// Mixed types work
	testSuccessful(t, "Math.min(i, 2, 3)", record, zuint64(1))
	testSuccessful(t, "Math.min(2, 3, i)", record, zint64(1))
	testSuccessful(t, "Math.max(i, 2, 3)", record, zuint64(3))
	testSuccessful(t, "Math.max(2, 3, i)", record, zint64(3))
	testSuccessful(t, "Math.min(1, -2.0)", record, zint64(-2))
	testSuccessful(t, "Math.min(-2.0, 1)", record, zfloat64(-2))
	testSuccessful(t, "Math.max(-1, 2.0)", record, zint64(2))
	testSuccessful(t, "Math.max(2.0, -1)", record, zfloat64(2))

	// Fails on invalid types
	testError(t, `Math.min("hello", 2)`, record, expr.ErrBadArgument, "min() on string")
	testError(t, `Math.max("hello", 2)`, record, expr.ErrBadArgument, "max() on string")
	testError(t, `Math.min(1.2.3.4, 2)`, record, expr.ErrBadArgument, "min() on ip")
	testError(t, `Math.max(1.2.3.4, 2)`, record, expr.ErrBadArgument, "max() on ip")

}
