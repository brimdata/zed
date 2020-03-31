package expr_test

import (
	"net"
	"strconv"
	"testing"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/stretchr/testify/require"
)

func zaddr(addr string) zng.Value {
	parsed := net.ParseIP(addr)
	return zng.Value{zng.TypeIP, zng.EncodeIP(parsed)}
}

func TestBadFunction(t *testing.T) {
	testError(t, "notafunction()", nil, expr.ErrNoSuchFunction, "calling nonexistent function")
}

func TestAbs(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[50;]`)
	require.NoError(t, err)

	testSuccessful(t, "Math.abs(-5)", record, zint64(5))
	testSuccessful(t, "Math.abs(5)", record, zint64(5))
	testSuccessful(t, "Math.abs(-3.2)", record, zfloat64(3.2))
	testSuccessful(t, "Math.abs(3.2)", record, zfloat64(3.2))
	testSuccessful(t, "Math.abs(u)", record, zuint64(50))

	testError(t, "Math.abs()", record, expr.ErrTooFewArgs, "abs with no args")
	testError(t, "Math.abs(1, 2)", record, expr.ErrTooManyArgs, "abs with too many args")
	testError(t, `Math.abs("hello")`, record, expr.ErrBadArgument, "abs with non-number")
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

func TestCeilFloorRound(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[50;]`)
	require.NoError(t, err)

	testSuccessful(t, "Math.ceil(1.5)", record, zfloat64(2))
	testSuccessful(t, "Math.floor(1.5)", record, zfloat64(1))
	testSuccessful(t, "Math.round(1.5)", record, zfloat64(2))

	testSuccessful(t, "Math.ceil(5)", record, zint64(5))
	testSuccessful(t, "Math.floor(5)", record, zint64(5))
	testSuccessful(t, "Math.round(5)", record, zint64(5))

	testError(t, "Math.ceil()", record, expr.ErrTooFewArgs, "ceil() with no args")
	testError(t, "Math.ceil(1, 2)", record, expr.ErrTooManyArgs, "ceil() with too many args")
	testError(t, "Math.floor()", record, expr.ErrTooFewArgs, "floor() with no args")
	testError(t, "Math.floor(1, 2)", record, expr.ErrTooManyArgs, "floor() with too many args")
	testError(t, "Math.round()", record, expr.ErrTooFewArgs, "round() with no args")
	testError(t, "Math.round(1, 2)", record, expr.ErrTooManyArgs, "round() with too many args")
}

func TestLogPow(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[50;]`)
	require.NoError(t, err)

	// Math.log() computes natural logarithm.  Rather than writing
	// out long floating point numbers in the parameters or results,
	// use more complex expressions that evaluate to simpler values.
	testSuccessful(t, "Math.log(32) / Math.log(2)", record, zfloat64(5))
	testSuccessful(t, "Math.log(32.0) / Math.log(2.0)", record, zfloat64(5))

	testSuccessful(t, "Math.pow(10, 2)", record, zfloat64(100))
	testSuccessful(t, "Math.pow(4.0, 1.5)", record, zfloat64(8))

	testError(t, "Math.log()", record, expr.ErrTooFewArgs, "log() with no args")
	testError(t, "Math.log(2, 3)", record, expr.ErrTooManyArgs, "log() with too many args")
	testError(t, "Math.log(0)", record, expr.ErrBadArgument, "log() of 0")
	testError(t, "Math.log(-1)", record, expr.ErrBadArgument, "log() of negative number")

	testError(t, "Math.pow()", record, expr.ErrTooFewArgs, "pow() with no args")
	testError(t, "Math.pow(2, 3, r)", record, expr.ErrTooManyArgs, "pow() with too many args")
	testError(t, "Math.pow(-1, 0.5)", record, expr.ErrBadArgument, "pow() with invalid arguments")
}

func TestMod(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[5;]`)
	require.NoError(t, err)

	testSuccessful(t, "Math.mod(5, 3)", record, zint64(2))
	testSuccessful(t, "Math.mod(u, 3)", record, zuint64(2))
	testSuccessful(t, "Math.mod(8, 5)", record, zint64(3))
	testSuccessful(t, "Math.mod(8, u)", record, zint64(3))

	testError(t, "Math.mod()", record, expr.ErrTooFewArgs, "mod() with no args")
	testError(t, "Math.mod(1, 2, 3)", record, expr.ErrTooManyArgs, "mod() with too many args")
	testError(t, "Math.mod(3.2, 2)", record, expr.ErrBadArgument, "mod() with float64 arg")
}

func TestStrFormat(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[5;]`)
	require.NoError(t, err)

	testSuccessful(t, "String.formatFloat(1.2)", record, zstring("1.2"))
	testError(t, "String.formatFloat()", record, expr.ErrTooFewArgs, "formatFloat() with no args")
	testError(t, "String.formatFloat(1.2, 3.4)", record, expr.ErrTooManyArgs, "formatFloat() with too many args")
	testError(t, "String.formatFloat(1)", record, expr.ErrBadArgument, "formatFloat() with non-float arg")

	testSuccessful(t, "String.formatInt(5)", record, zstring("5"))
	testError(t, "String.formatInt()", record, expr.ErrTooFewArgs, "formatInt() with no args")
	testError(t, "String.formatInt(3, 4)", record, expr.ErrTooManyArgs, "formatInt() with too many args")
	testError(t, "String.formatInt(1.5)", record, expr.ErrBadArgument, "formatInt() with non-int arg")

	testSuccessful(t, "String.formatIp(1.2.3.4)", record, zstring("1.2.3.4"))
	testError(t, "String.formatIp()", record, expr.ErrTooFewArgs, "formatIp() with no args")
	testError(t, "String.formatIp(1.2, 3.4)", record, expr.ErrTooManyArgs, "formatIp() with too many args")
	testError(t, "String.formatIp(1)", record, expr.ErrBadArgument, "formatIp() with non-ip arg")
}

func TestStrParse(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[5;]`)
	require.NoError(t, err)

	testSuccessful(t, `String.parseInt("1")`, record, zint64(1))
	testSuccessful(t, `String.parseInt("-1")`, record, zint64(-1))
	testError(t, `String.parseInt()`, record, expr.ErrTooFewArgs, "parseInt() with no args")
	testError(t, `String.parseInt("a", "b")`, record, expr.ErrTooManyArgs, "parseInt() with too many args")
	testError(t, `String.parseInt("abc")`, record, strconv.ErrSyntax, "parseInt() with non-parseable string")

	testSuccessful(t, `String.parseFloat("5.5")`, record, zfloat64(5.5))
	testError(t, `String.parseFloat()`, record, expr.ErrTooFewArgs, "parseFloat() with no args")
	testError(t, `String.parseFloat("a", "b")`, record, expr.ErrTooManyArgs, "parseFloat() with too many args")
	testError(t, `String.parseFloat("abc")`, record, strconv.ErrSyntax, "parseFloat() with non-parseable string")

	testSuccessful(t, `String.parseIp("1.2.3.4")`, record, zaddr("1.2.3.4"))
	testError(t, `String.parseIp()`, record, expr.ErrTooFewArgs, "parseIp() with no args")
	testError(t, `String.parseIp("a", "b")`, record, expr.ErrTooManyArgs, "parseIp() with too many args")
	testError(t, `String.parseIp("abc")`, record, expr.ErrBadArgument, "parseIp() with non-parseable string")
}

func TestOtherStrFuncs(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[u:uint64]
0:[5;]`)
	require.NoError(t, err)

	testSuccessful(t, `String.replace("bann", "n", "na")`, record, zstring("banana"))
	testError(t, `String.replace("foo", "bar")`, record, expr.ErrTooFewArgs, "replace() with too few args")
	testError(t, `String.replace("foo", "bar", "baz", "blort")`, record, expr.ErrTooManyArgs, "replace() with too many args")
	testError(t, `String.replace("foo", "o", 5)`, record, expr.ErrBadArgument, "replace() with non-string arg")

	testSuccessful(t, `String.toLower("BOO")`, record, zstring("boo"))
	testError(t, `String.toLower()`, record, expr.ErrTooFewArgs, "toLower() with no args")
	testError(t, `String.toLower("BOO", "HOO")`, record, expr.ErrTooManyArgs, "toLower() with too many args")

	testSuccessful(t, `String.toUpper("boo")`, record, zstring("BOO"))
	testError(t, `String.toUpper()`, record, expr.ErrTooFewArgs, "toUpper() with no args")
	testError(t, `String.toUpper("boo", "hoo")`, record, expr.ErrTooManyArgs, "toUpper() with too many args")

	testSuccessful(t, `String.trim("  hi  there   ")`, record, zstring("hi  there"))
	testError(t, `String.trim()`, record, expr.ErrTooFewArgs, "trim() with no args")
	testError(t, `String.trim("  hi  ", "  there  ")`, record, expr.ErrTooManyArgs, "trim() with too many args")
}

func TestLen(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[s:set[int32],a:array[int32]]
0:[[1;2;3;][4;5;6;]]`)
	require.NoError(t, err)

	testSuccessful(t, "len(s)", record, zint64(3))
	testSuccessful(t, "len(a)", record, zint64(3))

	testError(t, "len()", record, expr.ErrTooFewArgs, "len() with no args")
	testError(t, `len("foo", "bar")`, record, expr.ErrTooManyArgs, "len() with too many args")
	testError(t, `len("foo")`, record, expr.ErrBadArgument, "len() with string arg")
	testError(t, "len(5)", record, expr.ErrBadArgument, "len() with non-container arg")

	record, err = parseOneRecord(`
#0:record[s:string,bs:bstring,bs2:bstring]
0:[🍺;\xf0\x9f\x8d\xba;\xba\x8d\x9f\xf0;]`)
	require.NoError(t, err)

	testSuccessful(t, `String.byteLen("foo")`, record, zint64(3))
	testSuccessful(t, `String.byteLen(s)`, record, zint64(4))
	testSuccessful(t, `String.byteLen(bs)`, record, zint64(4))
	testSuccessful(t, `String.byteLen(bs2)`, record, zint64(4))

	testSuccessful(t, `String.runeLen("foo")`, record, zint64(3))
	testSuccessful(t, `String.runeLen(s)`, record, zint64(1))
	testSuccessful(t, `String.runeLen(bs)`, record, zint64(1))
	testSuccessful(t, `String.runeLen(bs2)`, record, zint64(4))
}
