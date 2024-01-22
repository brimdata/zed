package expr_test

import (
	"errors"
	"testing"

	"github.com/brimdata/zed/runtime/sam/expr/function"
)

func TestBadFunction(t *testing.T) {
	testError(t, "notafunction()", function.ErrNoSuchFunction, "calling nonexistent function")
}

func TestAbs(t *testing.T) {
	const record = "{u:50 (uint64)} (=0)"

	testSuccessful(t, "abs(-5)", record, "5")
	testSuccessful(t, "abs(5)", record, "5")
	testSuccessful(t, "abs(-3.2)", record, "3.2")
	testSuccessful(t, "abs(3.2)", record, "3.2")
	testSuccessful(t, "abs(u)", record, "50(uint64)")

	testError(t, "abs()", function.ErrTooFewArgs, "abs with no args")
	testError(t, "abs(1, 2)", function.ErrTooManyArgs, "abs with too many args")
	testSuccessful(t, `abs("hello")`, record, `error({message:"abs: not a number",on:"hello"})`)
}

func TestSqrt(t *testing.T) {
	const record = "{f:6.25,i:9 (int32)} (=0)"

	testSuccessful(t, "sqrt(4.0)", record, "2.")
	testSuccessful(t, "sqrt(f)", record, "2.5")
	testSuccessful(t, "sqrt(i)", record, "3.")

	testError(t, "sqrt()", function.ErrTooFewArgs, "sqrt with no args")
	testError(t, "sqrt(1, 2)", function.ErrTooManyArgs, "sqrt with too many args")
	testSuccessful(t, "sqrt(-1)", record, "NaN")
}

func TestMinMax(t *testing.T) {
	const record = "{i:1 (uint64),f:2.} (=0)"

	// Simple cases
	testSuccessful(t, "min(1, 2, 3)", record, "1")
	testSuccessful(t, "max(1, 2, 3)", record, "3")
	testSuccessful(t, "min(3, 2, 1)", record, "1")
	testSuccessful(t, "max(3, 2, 1)", record, "3")

	// Mixed types work
	testSuccessful(t, "min(i, 2, 3)", record, "1(uint64)")
	testSuccessful(t, "min(2, 3, i)", record, "1")
	testSuccessful(t, "max(i, 2, 3)", record, "3(uint64)")
	testSuccessful(t, "max(2, 3, i)", record, "3")
	testSuccessful(t, "min(1, -2.0)", record, "-2")
	testSuccessful(t, "min(-2.0, 1)", record, "-2.")
	testSuccessful(t, "max(-1, 2.0)", record, "2")
	testSuccessful(t, "max(2.0, -1)", record, "2.")

	// Fails on invalid types
	testSuccessful(t, `min("hello", 2)`, record, `error({message:"min: not a number",on:"hello"})`)
	testSuccessful(t, `max("hello", 2)`, record, `error({message:"max: not a number",on:"hello"})`)
	testSuccessful(t, `min(1.2.3.4, 2)`, record, `error({message:"min: not a number",on:1.2.3.4})`)
	testSuccessful(t, `max(1.2.3.4, 2)`, record, `error({message:"max: not a number",on:1.2.3.4})`)
}

func TestCeilFloorRound(t *testing.T) {
	testSuccessful(t, "ceil(1.5)", "", "2.")
	testSuccessful(t, "floor(1.5)", "", "1.")
	testSuccessful(t, "round(1.5)", "", "2.")

	testSuccessful(t, "ceil(5)", "", "5")
	testSuccessful(t, "floor(5)", "", "5")
	testSuccessful(t, "round(5)", "", "5")

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
	testSuccessful(t, "log(32) / log(2)", "", "5.")
	testSuccessful(t, "log(32.0) / log(2.0)", "", "5.")

	testSuccessful(t, "pow(10, 2)", "", "100.")
	testSuccessful(t, "pow(4.0, 1.5)", "", "8.")

	testError(t, "log()", function.ErrTooFewArgs, "log() with no args")
	testError(t, "log(2, 3)", function.ErrTooManyArgs, "log() with too many args")
	testSuccessful(t, "log(0)", "", `error({message:"log: illegal argument",on:0})`)
	testSuccessful(t, "log(-1)", "", `error({message:"log: illegal argument",on:-1})`)

	testError(t, "pow()", function.ErrTooFewArgs, "pow() with no args")
	testError(t, "pow(2, 3, r)", function.ErrTooManyArgs, "pow() with too many args")
	testSuccessful(t, "pow(-1, 0.5)", "", "NaN")
}

func TestOtherStrFuncs(t *testing.T) {
	testSuccessful(t, `replace("bann", "n", "na")`, "", `"banana"`)
	testError(t, `replace("foo", "bar")`, function.ErrTooFewArgs, "replace() with too few args")
	testError(t, `replace("foo", "bar", "baz", "blort")`, function.ErrTooManyArgs, "replace() with too many args")
	testSuccessful(t, `replace("foo", "o", 5)`, "", `error({message:"replace: string arg required",on:5})`)

	testSuccessful(t, `lower("BOO")`, "", `"boo"`)
	testError(t, `lower()`, function.ErrTooFewArgs, "toLower() with no args")
	testError(t, `lower("BOO", "HOO")`, function.ErrTooManyArgs, "toLower() with too many args")

	testSuccessful(t, `upper("boo")`, "", `"BOO"`)
	testError(t, `upper()`, function.ErrTooFewArgs, "toUpper() with no args")
	testError(t, `upper("boo", "hoo")`, function.ErrTooManyArgs, "toUpper() with too many args")

	testSuccessful(t, `trim("  hi  there   ")`, "", `"hi  there"`)
	testError(t, `trim()`, function.ErrTooFewArgs, "trim() with no args")
	testError(t, `trim("  hi  ", "  there  ")`, function.ErrTooManyArgs, "trim() with too many args")
}

func TestLen(t *testing.T) {
	record := "{s:|[1 (int32),2 (int32),3 (int32)]| (=0),a:[4 (int32),5 (int32),6 (int32)] (=1)} (=2)"

	testSuccessful(t, "len(s)", record, "3")
	testSuccessful(t, "len(a)", record, "3")

	testError(t, "len()", function.ErrTooFewArgs, "len() with no args")
	testError(t, `len("foo", "bar")`, function.ErrTooManyArgs, "len() with too many args")
	testSuccessful(t, "len(5)", record, `error({message:"len: bad type",on:5})`)

	record = `{s:"🍺",bs:0xf09f8dba}`

	testSuccessful(t, `len("foo")`, record, "3")
	testSuccessful(t, `len(s)`, record, "4")
	testSuccessful(t, `len(bs)`, record, "4")

	testSuccessful(t, `rune_len("foo")`, record, "3")
	testSuccessful(t, `rune_len(s)`, record, "1")
}

func TestCast(t *testing.T) {
	// Constant type argument
	testSuccessful(t, "cast(1, <uint64>)", "", "1(uint64)")
	testError(t, "cast(1, 2)", errors.New("shaper type argument is not a type: 2"), "cast() argument is not a type")

	// Constant name argument
	testSuccessful(t, `cast(1, "my_int64")`, "", "1(=my_int64)")
	testError(t, `cast(1, "uint64")`, errors.New(`bad type name "uint64": primitive type name`), "cast() argument is a primitve type name")

	// Variable type argument
	testSuccessful(t, "cast(1, type)", "{type:<uint64>}", "1(uint64)")
	testSuccessful(t, "cast(1, type)", "{type:2}", `error({message:"shaper type argument is not a type",on:2})`)

	// Variable name argument
	testSuccessful(t, "cast(1, name)", `{name:"my_int64"}`, "1(=my_int64)")
	testSuccessful(t, "cast(1, name)", `{name:"uint64"}`, `error("bad type name \"uint64\": primitive type name")`)

	testError(t, "cast()", function.ErrTooFewArgs, "cast() with no args")
	testError(t, "cast(1, 2, 3)", function.ErrTooManyArgs, "cast() with no args")
}
