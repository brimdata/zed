package function_test

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

	testError(t, "Math.sqrt()", record, expr.ErrWrongArgc, "sqrt with no args")
	testError(t, "Math.sqrt(1, 2)", record, expr.ErrWrongArgc, "sqrt with too many args")
	testError(t, "Math.sqrt(-1)", record, expr.ErrBadArgument, "sqrt of negative")
}
