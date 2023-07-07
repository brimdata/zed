package expr_test

import (
	"fmt"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/function"
	"github.com/brimdata/zed/zson"
)

func TestBadFunction(t *testing.T) {
	testError(t, "notafunction()", function.ErrNoSuchFunction, "calling nonexistent function")
}

func ZSON(s string) zed.Value {
	val, err := zson.ParseValue(zed.NewContext(), s)
	if err != nil {
		panic(fmt.Sprintf("zson parse failed compiling: %q (%s)", s, err))
	}
	return *val
}

func TestAbs(t *testing.T) {
	const record = "{u:50 (uint64)} (=0)"

	testSuccessful(t, "abs(-5)", record, *zed.NewInt64(5))
	testSuccessful(t, "abs(5)", record, *zed.NewInt64(5))
	testSuccessful(t, "abs(-3.2)", record, *zed.NewFloat64(3.2))
	testSuccessful(t, "abs(3.2)", record, *zed.NewFloat64(3.2))
	testSuccessful(t, "abs(u)", record, *zed.NewUint64(50))

	testError(t, "abs()", function.ErrTooFewArgs, "abs with no args")
	testError(t, "abs(1, 2)", function.ErrTooManyArgs, "abs with too many args")
	testSuccessful(t, `abs("hello")`, record, ZSON(`error({message:"abs: not a number", on:"hello"})`))
}

func TestSqrt(t *testing.T) {
	const record = "{f:6.25,i:9 (int32)} (=0)"

	testSuccessful(t, "sqrt(4.0)", record, *zed.NewFloat64(2.0))
	testSuccessful(t, "sqrt(f)", record, *zed.NewFloat64(2.5))
	testSuccessful(t, "sqrt(i)", record, *zed.NewFloat64(3.0))

	testError(t, "sqrt()", function.ErrTooFewArgs, "sqrt with no args")
	testError(t, "sqrt(1, 2)", function.ErrTooManyArgs, "sqrt with too many args")
	testSuccessful(t, "sqrt(-1)", record, ZSON("NaN"))
}

func TestMinMax(t *testing.T) {
	const record = "{i:1 (uint64),f:2.} (=0)"

	// Simple cases
	testSuccessful(t, "min(1, 2, 3)", record, *zed.NewInt64(1))
	testSuccessful(t, "max(1, 2, 3)", record, *zed.NewInt64(3))
	testSuccessful(t, "min(3, 2, 1)", record, *zed.NewInt64(1))
	testSuccessful(t, "max(3, 2, 1)", record, *zed.NewInt64(3))

	// Mixed types work
	testSuccessful(t, "min(i, 2, 3)", record, *zed.NewUint64(1))
	testSuccessful(t, "min(2, 3, i)", record, *zed.NewInt64(1))
	testSuccessful(t, "max(i, 2, 3)", record, *zed.NewUint64(3))
	testSuccessful(t, "max(2, 3, i)", record, *zed.NewInt64(3))
	testSuccessful(t, "min(1, -2.0)", record, *zed.NewInt64(-2))
	testSuccessful(t, "min(-2.0, 1)", record, *zed.NewFloat64(-2))
	testSuccessful(t, "max(-1, 2.0)", record, *zed.NewInt64(2))
	testSuccessful(t, "max(2.0, -1)", record, *zed.NewFloat64(2))

	// Fails on invalid types
	testSuccessful(t, `min("hello", 2)`, record, ZSON(`error({message:"min: not a number",on:"hello"})`))
	testSuccessful(t, `max("hello", 2)`, record, ZSON(`error({message:"max: not a number",on:"hello"})`))
	testSuccessful(t, `min(1.2.3.4, 2)`, record, ZSON(`error({message:"min: not a number",on:1.2.3.4})`))
	testSuccessful(t, `max(1.2.3.4, 2)`, record, ZSON(`error({message:"max: not a number",on:1.2.3.4})`))
}

func TestCeilFloorRound(t *testing.T) {
	testSuccessful(t, "ceil(1.5)", "", *zed.NewFloat64(2))
	testSuccessful(t, "floor(1.5)", "", *zed.NewFloat64(1))
	testSuccessful(t, "round(1.5)", "", *zed.NewFloat64(2))

	testSuccessful(t, "ceil(5)", "", *zed.NewInt64(5))
	testSuccessful(t, "floor(5)", "", *zed.NewInt64(5))
	testSuccessful(t, "round(5)", "", *zed.NewInt64(5))

	testError(t, "ceil()", function.ErrTooFewArgs, "ceil() with no args")
	testError(t, "ceil(1, 2)", function.ErrTooManyArgs, "ceil() with too many args")
	testError(t, "floor()", function.ErrTooFewArgs, "floor() with no args")
	testError(t, "floor(1, 2)", function.ErrTooManyArgs, "floor() with too many args")
	testError(t, "round()", function.ErrTooFewArgs, "round() with no args")
	testError(t, "round(1, 2)", function.ErrTooManyArgs, "round() with too many args")
}

func TestLogPow(t *testing.T) {
	// Math.log() computes natural logarithm.  Rather than writing
	// out long floating point numbers in the parameters or results,
	// use more complex expressions that evaluate to simpler values.
	testSuccessful(t, "log(32) / log(2)", "", *zed.NewFloat64(5))
	testSuccessful(t, "log(32.0) / log(2.0)", "", *zed.NewFloat64(5))

	testSuccessful(t, "pow(10, 2)", "", *zed.NewFloat64(100))
	testSuccessful(t, "pow(4.0, 1.5)", "", *zed.NewFloat64(8))

	testError(t, "log()", function.ErrTooFewArgs, "log() with no args")
	testError(t, "log(2, 3)", function.ErrTooManyArgs, "log() with too many args")
	testSuccessful(t, "log(0)", "", ZSON(`error({message:"log: illegal argument",on:0})`))
	testSuccessful(t, "log(-1)", "", ZSON(`error({message:"log: illegal argument",on:-1})`))

	testError(t, "pow()", function.ErrTooFewArgs, "pow() with no args")
	testError(t, "pow(2, 3, r)", function.ErrTooManyArgs, "pow() with too many args")
	testSuccessful(t, "pow(-1, 0.5)", "", ZSON("NaN"))
}

func TestOtherStrFuncs(t *testing.T) {
	testSuccessful(t, `replace("bann", "n", "na")`, "", *zed.NewString("banana"))
	testError(t, `replace("foo", "bar")`, function.ErrTooFewArgs, "replace() with too few args")
	testError(t, `replace("foo", "bar", "baz", "blort")`, function.ErrTooManyArgs, "replace() with too many args")
	testSuccessful(t, `replace("foo", "o", 5)`, "", ZSON(`error({message:"replace: string arg required",on:5})`))

	testSuccessful(t, `lower("BOO")`, "", *zed.NewString("boo"))
	testError(t, `lower()`, function.ErrTooFewArgs, "toLower() with no args")
	testError(t, `lower("BOO", "HOO")`, function.ErrTooManyArgs, "toLower() with too many args")

	testSuccessful(t, `upper("boo")`, "", *zed.NewString("BOO"))
	testError(t, `upper()`, function.ErrTooFewArgs, "toUpper() with no args")
	testError(t, `upper("boo", "hoo")`, function.ErrTooManyArgs, "toUpper() with too many args")

	testSuccessful(t, `trim("  hi  there   ")`, "", *zed.NewString("hi  there"))
	testError(t, `trim()`, function.ErrTooFewArgs, "trim() with no args")
	testError(t, `trim("  hi  ", "  there  ")`, function.ErrTooManyArgs, "trim() with too many args")
}

func TestLen(t *testing.T) {
	record := "{s:|[1 (int32),2 (int32),3 (int32)]| (=0),a:[4 (int32),5 (int32),6 (int32)] (=1)} (=2)"

	testSuccessful(t, "len(s)", record, *zed.NewInt64(3))
	testSuccessful(t, "len(a)", record, *zed.NewInt64(3))

	testError(t, "len()", function.ErrTooFewArgs, "len() with no args")
	testError(t, `len("foo", "bar")`, function.ErrTooManyArgs, "len() with too many args")
	testSuccessful(t, "len(5)", record, ZSON(`error({message:"len: bad type",on:5})`))

	record = `{s:"🍺",bs:0xf09f8dba}`

	testSuccessful(t, `len("foo")`, record, *zed.NewInt64(3))
	testSuccessful(t, `len(s)`, record, *zed.NewInt64(4))
	testSuccessful(t, `len(bs)`, record, *zed.NewInt64(4))

	testSuccessful(t, `rune_len("foo")`, record, *zed.NewInt64(3))
	testSuccessful(t, `rune_len(s)`, record, *zed.NewInt64(1))
}
