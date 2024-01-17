package expr_test

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/ztest"
	"github.com/x448/float16"
)

func testSuccessful(t *testing.T, e string, input string, expectedVal zed.Value) {
	t.Helper()
	if input == "" {
		input = "{}"
	}
	zt := ztest.ZTest{
		Zed:    fmt.Sprintf("yield %s", e),
		Input:  input,
		Output: zson.FormatValue(expectedVal) + "\n",
	}
	if err := zt.RunInternal(""); err != nil {
		t.Fatal(err)
	}
}

func testError(t *testing.T, e string, expectErr error, description string) {
	t.Helper()
	zt := ztest.ZTest{
		Zed:     fmt.Sprintf("yield %s", e),
		ErrorRE: expectErr.Error(),
	}
	if err := zt.RunInternal(""); err != nil {
		t.Fatal(err)
	}
}

func TestPrimitives(t *testing.T) {
	const record = `{x:10 (int32),f:2.5,s:"hello"} (=0)`

	// Test simple literals
	testSuccessful(t, "50", record, zed.NewInt64(50))
	testSuccessful(t, "3.14", record, zed.NewFloat64(3.14))
	testSuccessful(t, `"boo"`, record, zed.NewString("boo"))

	// Test good field references
	testSuccessful(t, "x", record, zed.NewInt32(10))
	testSuccessful(t, "f", record, zed.NewFloat64(2.5))
	testSuccessful(t, "s", record, zed.NewString("hello"))
}

func TestCompareNumbers(t *testing.T) {
	var numericTypes = []string{
		"uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float16", "float32", "float64"}
	var intFields = []string{"u8", "i16", "u16", "i32", "u32", "i64", "u64"}

	for _, typ := range numericTypes {
		// Make a test point with this in a field called x plus
		// one field of each other integer type
		one := "1"
		if strings.HasPrefix(typ, "float") {
			one = "1."
		}
		record := fmt.Sprintf(
			"{x:%s (%s),u8:0 (uint8),i16:0 (int16),u16:0 (uint16),i32:0 (int32),u32:0 (uint32),i64:0,u64:0 (uint64)} (=0)",
			one, typ)
		// Test the 6 comparison operators against a constant
		testSuccessful(t, "x == 1", record, zed.True)
		testSuccessful(t, "x == 0", record, zed.False)
		testSuccessful(t, "x != 0", record, zed.True)
		testSuccessful(t, "x != 1", record, zed.False)
		testSuccessful(t, "x < 2", record, zed.True)
		testSuccessful(t, "x < 1", record, zed.False)
		testSuccessful(t, "x <= 2", record, zed.True)
		testSuccessful(t, "x <= 1", record, zed.True)
		testSuccessful(t, "x <= 0", record, zed.False)
		testSuccessful(t, "x > 0", record, zed.True)
		testSuccessful(t, "x > 1", record, zed.False)
		testSuccessful(t, "x >= 0", record, zed.True)
		testSuccessful(t, "x >= 1", record, zed.True)
		testSuccessful(t, "x >= 2", record, zed.False)

		// Test the full matrix of comparisons between all
		// the integer types
		for _, other := range intFields {
			exp := fmt.Sprintf("x == %s", other)
			testSuccessful(t, exp, record, zed.False)

			exp = fmt.Sprintf("x != %s", other)
			testSuccessful(t, exp, record, zed.True)

			exp = fmt.Sprintf("x < %s", other)
			testSuccessful(t, exp, record, zed.False)

			exp = fmt.Sprintf("x <= %s", other)
			testSuccessful(t, exp, record, zed.False)

			exp = fmt.Sprintf("x > %s", other)
			testSuccessful(t, exp, record, zed.True)

			exp = fmt.Sprintf("x >= %s", other)
			testSuccessful(t, exp, record, zed.True)
		}

		// For integer types, test this against other
		// number-ish types: port, time, duration
		if !strings.HasPrefix(typ, "float") {
			record := fmt.Sprintf(
				"{x:%s (%s),p:80 (port=uint16),t:2020-03-09T22:54:12Z,d:16m40s}", one, typ)

			// port
			testSuccessful(t, "x == p", record, zed.False)
			testSuccessful(t, "p == x", record, zed.False)
			testSuccessful(t, "x != p", record, zed.True)
			testSuccessful(t, "p != x", record, zed.True)
			testSuccessful(t, "x < p", record, zed.True)
			testSuccessful(t, "p < x", record, zed.False)
			testSuccessful(t, "x <= p", record, zed.True)
			testSuccessful(t, "p <= x", record, zed.False)
			testSuccessful(t, "x > p", record, zed.False)
			testSuccessful(t, "p > x", record, zed.True)
			testSuccessful(t, "x >= p", record, zed.False)
			testSuccessful(t, "p >= x", record, zed.True)

			// time
			testSuccessful(t, "x == t", record, zed.False)
			testSuccessful(t, "t == x", record, zed.False)
			testSuccessful(t, "x != t", record, zed.True)
			testSuccessful(t, "t != x", record, zed.True)
			testSuccessful(t, "x < t", record, zed.True)
			testSuccessful(t, "t < x", record, zed.False)
			testSuccessful(t, "x <= t", record, zed.True)
			testSuccessful(t, "t <= x", record, zed.False)
			testSuccessful(t, "x > t", record, zed.False)
			testSuccessful(t, "t > x", record, zed.True)
			testSuccessful(t, "x >= t", record, zed.False)
			testSuccessful(t, "t >= x", record, zed.True)

			// duration
			testSuccessful(t, "x == d", record, zed.False)
			testSuccessful(t, "d == x", record, zed.False)
			testSuccessful(t, "x != d", record, zed.True)
			testSuccessful(t, "d != x", record, zed.True)
			testSuccessful(t, "x < d", record, zed.True)
			testSuccessful(t, "d < x", record, zed.False)
			testSuccessful(t, "x <= d", record, zed.True)
			testSuccessful(t, "d <= x", record, zed.False)
			testSuccessful(t, "x > d", record, zed.False)
			testSuccessful(t, "d > x", record, zed.True)
			testSuccessful(t, "x >= d", record, zed.False)
			testSuccessful(t, "d >= x", record, zed.True)
		}

		// Test this against non-numeric types
		record = fmt.Sprintf(
			`{x:%s (%s),s:"hello",i:10.1.1.1,n:10.1.0.0/16} (=0)`, one, typ)

		testSuccessful(t, "x == s", record, ZSON("false"))
		testSuccessful(t, "x != s", record, ZSON("true"))
		testSuccessful(t, "x < s", record, ZSON("false"))
		testSuccessful(t, "x <= s", record, ZSON("false"))
		testSuccessful(t, "x > s", record, ZSON("false"))
		testSuccessful(t, "x >= s", record, ZSON("false"))

		testSuccessful(t, "x == i", record, ZSON("false"))
		testSuccessful(t, "x != i", record, ZSON("true"))
		testSuccessful(t, "x < i", record, ZSON("false"))
		testSuccessful(t, "x <= i", record, ZSON("false"))
		testSuccessful(t, "x > i", record, ZSON("false"))
		testSuccessful(t, "x >= i", record, ZSON("false"))

		testSuccessful(t, "x == n", record, ZSON("false"))
		testSuccessful(t, "x != n", record, ZSON("true"))
		testSuccessful(t, "x < n", record, ZSON("false"))
		testSuccessful(t, "x <= n", record, ZSON("false"))
		testSuccessful(t, "x > n", record, ZSON("false"))
		testSuccessful(t, "x >= n", record, ZSON("false"))
	}

	// Test comparison between signed and unsigned and also
	// floats that cast to different integers.
	const rec2 = "{i:-1,u:18446744073709551615 (uint64),f:-1.} (=0)"

	testSuccessful(t, "i == u", rec2, zed.False)
	testSuccessful(t, "i != u", rec2, zed.True)
	testSuccessful(t, "i < u", rec2, zed.True)
	testSuccessful(t, "i <= u", rec2, zed.True)
	testSuccessful(t, "i > u", rec2, zed.False)
	testSuccessful(t, "i >= u", rec2, zed.False)

	testSuccessful(t, "u == i", rec2, zed.False)
	testSuccessful(t, "u != i", rec2, zed.True)
	testSuccessful(t, "u < i", rec2, zed.False)
	testSuccessful(t, "u <= i", rec2, zed.False)
	testSuccessful(t, "u > i", rec2, zed.True)
	testSuccessful(t, "u >= i", rec2, zed.True)

	testSuccessful(t, "f == u", rec2, zed.False)
	testSuccessful(t, "f != u", rec2, zed.True)
	testSuccessful(t, "f < u", rec2, zed.True)
	testSuccessful(t, "f <= u", rec2, zed.True)
	testSuccessful(t, "f > u", rec2, zed.False)
	testSuccessful(t, "f >= u", rec2, zed.False)

	testSuccessful(t, "u == f", rec2, zed.False)
	testSuccessful(t, "u != f", rec2, zed.True)
	testSuccessful(t, "u < f", rec2, zed.False)
	testSuccessful(t, "u <= f", rec2, zed.False)
	testSuccessful(t, "u > f", rec2, zed.True)
	testSuccessful(t, "u >= f", rec2, zed.True)
}

func TestCompareNonNumbers(t *testing.T) {
	record := `
{
    b: true,
    s: "hello",
    i: 10.1.1.1,
    p: 443 (port=uint16),
    net: 10.1.0.0/16,
    t: 2020-03-09T22:54:12Z,
    d: 16m40s
} (=0)
`

	// bool
	testSuccessful(t, "b == true", record, zed.True)
	testSuccessful(t, "b == false", record, zed.False)
	testSuccessful(t, "b != true", record, zed.False)
	testSuccessful(t, "b != false", record, zed.True)

	// string
	testSuccessful(t, `s == "hello"`, record, zed.True)
	testSuccessful(t, `s != "hello"`, record, zed.False)
	testSuccessful(t, `s == "world"`, record, zed.False)
	testSuccessful(t, `s != "world"`, record, zed.True)

	// ip
	testSuccessful(t, "i == 10.1.1.1", record, zed.True)
	testSuccessful(t, "i != 10.1.1.1", record, zed.False)
	testSuccessful(t, "i == 1.1.1.10", record, zed.False)
	testSuccessful(t, "i != 1.1.1.10", record, zed.True)
	testSuccessful(t, "i == i", record, zed.True)

	// port
	testSuccessful(t, "p == 443", record, zed.True)
	testSuccessful(t, "p != 443", record, zed.False)

	// net
	testSuccessful(t, "net == 10.1.0.0/16", record, zed.True)
	testSuccessful(t, "net != 10.1.0.0/16", record, zed.False)
	testSuccessful(t, "net == 10.1.0.0/24", record, zed.False)
	testSuccessful(t, "net != 10.1.0.0/24", record, zed.True)

	// Test comparisons between incompatible types
	allTypes := []struct {
		field string
		typ   string
	}{
		{"b", "bool"},
		{"s", "string"},
		{"i", "ip"},
		{"p", "port"},
		{"net", "net"},
	}

	allOperators := []string{"==", "!=", "<", "<=", ">", ">="}

	for _, t1 := range allTypes {
		for _, t2 := range allTypes {
			if t1 == t2 {
				continue
			}
			for _, op := range allOperators {
				exp := fmt.Sprintf("%s %s %s", t1.field, op, t2.field)
				// XXX we no longer have a way to
				// propagate boolean "warnings"...
				testSuccessful(t, exp, record, zed.NewBool(op == "!="))
			}
		}
	}

	// relative comparisons on strings
	record = `{s:"abc"}`

	testSuccessful(t, `s < "brim"`, record, zed.True)
	testSuccessful(t, `s < "aaa"`, record, zed.False)
	testSuccessful(t, `s < "abc"`, record, zed.False)

	testSuccessful(t, `s > "brim"`, record, zed.False)
	testSuccessful(t, `s > "aaa"`, record, zed.True)
	testSuccessful(t, `s > "abc"`, record, zed.False)

	testSuccessful(t, `s <= "brim"`, record, zed.True)
	testSuccessful(t, `s <= "aaa"`, record, zed.False)
	testSuccessful(t, `s <= "abc"`, record, zed.True)

	testSuccessful(t, `s >= "brim"`, record, zed.False)
	testSuccessful(t, `s >= "aaa"`, record, zed.True)
	testSuccessful(t, `s >= "abc"`, record, zed.True)
}

func TestPattern(t *testing.T) {
	testSuccessful(t, `"abc" == "abc"`, "", zed.True)
	testSuccessful(t, `"abc" != "abc"`, "", zed.False)
	testSuccessful(t, "cidr_match(10.0.0.0/8, 10.1.1.1)", "", zed.True)
	testSuccessful(t, "10.1.1.1 in 192.168.0.0/16", "", zed.False)
	testSuccessful(t, "!cidr_match(10.0.0.0/8, 10.1.1.1)", "", zed.False)
	testSuccessful(t, "!(10.1.1.1 in 192.168.0.0/16)", "", zed.True)
}

func TestIn(t *testing.T) {
	const record = "{a:[1 (int32),2 (int32),3 (int32)] (=0),s:|[4 (int32),5 (int32),6 (int32)]| (=1)} (=2)"

	testSuccessful(t, "1 in a", record, zed.True)
	testSuccessful(t, "0 in a", record, zed.False)

	testSuccessful(t, "1 in s", record, zed.False)
	testSuccessful(t, "4 in s", record, zed.True)

	testSuccessful(t, `"boo" in a`, record, zed.False)
	testSuccessful(t, `"boo" in s`, record, zed.False)
}

func TestArithmetic(t *testing.T) {
	record := "{x:10 (int32),f:2.5} (=0)"

	// Test integer arithmetic
	testSuccessful(t, "100 + 23", record, zed.NewInt64(123))
	testSuccessful(t, "x + 5", record, zed.NewInt64(15))
	testSuccessful(t, "5 + x", record, zed.NewInt64(15))
	testSuccessful(t, "x - 5", record, zed.NewInt64(5))
	testSuccessful(t, "0 - x", record, zed.NewInt64(-10))
	testSuccessful(t, "x + 5 - 3", record, zed.NewInt64(12))
	testSuccessful(t, "x*2", record, zed.NewInt64(20))
	testSuccessful(t, "5*x*2", record, zed.NewInt64(100))
	testSuccessful(t, "x/3", record, zed.NewInt64(3))
	testSuccessful(t, "20/x", record, zed.NewInt64(2))

	// Test precedence of arithmetic operations
	testSuccessful(t, "x + 1 * 10", record, zed.NewInt64(20))
	testSuccessful(t, "(x + 1) * 10", record, zed.NewInt64(110))

	// Test arithmetic with floats
	testSuccessful(t, "f + 1.0", record, zed.NewFloat64(3.5))
	testSuccessful(t, "1.0 + f", record, zed.NewFloat64(3.5))
	testSuccessful(t, "f - 1.0", record, zed.NewFloat64(1.5))
	testSuccessful(t, "0.0 - f", record, zed.NewFloat64(-2.5))
	testSuccessful(t, "f * 1.5", record, zed.NewFloat64(3.75))
	testSuccessful(t, "1.5 * f", record, zed.NewFloat64(3.75))
	testSuccessful(t, "f / 1.25", record, zed.NewFloat64(2.0))
	testSuccessful(t, "5.0 / f", record, zed.NewFloat64(2.0))

	// Difference of two times is a duration
	testSuccessful(t, "a - b", "{a:2022-09-22T00:00:01Z,b:2022-09-22T00:00:00Z}",
		zed.NewDuration(nano.Second))

	width := func(id int) int {
		switch id {
		case zed.IDInt8, zed.IDUint8:
			return 8
		case zed.IDInt16, zed.IDUint16:
			return 16
		case zed.IDInt32, zed.IDUint32:
			return 32
		case zed.IDInt64, zed.IDUint64:
			return 64
		}
		panic("width")
	}
	signed := func(width int) zed.Type {
		switch width {
		case 8:
			return zed.TypeInt8
		case 16:
			return zed.TypeInt16
		case 32:
			return zed.TypeInt32
		case 64:
			return zed.TypeInt64
		}
		panic("signed")
	}
	unsigned := func(width int) zed.Type {
		switch width {
		case 8:
			return zed.TypeUint8
		case 16:
			return zed.TypeUint16
		case 32:
			return zed.TypeUint32
		case 64:
			return zed.TypeUint64
		}
		panic("signed")
	}
	// Test arithmetic between integer types
	iresult := func(t1, t2 string, v uint64) zed.Value {
		typ1 := zed.LookupPrimitive(t1)
		typ2 := zed.LookupPrimitive(t2)
		id1 := typ1.ID()
		id2 := typ2.ID()
		sign1 := zed.IsSigned(id1)
		sign2 := zed.IsSigned(id2)
		sign := true
		if sign1 == sign2 {
			sign = sign1
		}
		w := width(id1)
		if w2 := width(id2); w2 > w {
			w = w2
		}
		if sign {
			return zed.NewInt(signed(w), int64(v))
		}
		return zed.NewUint(unsigned(w), v)
	}

	var intTypes = []string{"int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64"}
	for _, t1 := range intTypes {
		for _, t2 := range intTypes {
			record := fmt.Sprintf("{a:4 (%s),b:2 (%s)} (=0)", t1, t2)
			testSuccessful(t, "a + b", record, iresult(t1, t2, 6))
			testSuccessful(t, "b + a", record, iresult(t1, t2, 6))
			testSuccessful(t, "a - b", record, iresult(t1, t2, 2))
			testSuccessful(t, "a * b", record, iresult(t1, t2, 8))
			testSuccessful(t, "b * a", record, iresult(t1, t2, 8))
			testSuccessful(t, "a / b", record, iresult(t1, t2, 2))
			testSuccessful(t, "b / a", record, iresult(t1, t2, 0))
		}

		// Test arithmetic mixing float + int
		record = fmt.Sprintf("{x:10 (%s),f:2.5} (=0)", t1)
		testSuccessful(t, "f + 5", record, zed.NewFloat64(7.5))
		testSuccessful(t, "5 + f", record, zed.NewFloat64(7.5))
		testSuccessful(t, "f + x", record, zed.NewFloat64(12.5))
		testSuccessful(t, "x + f", record, zed.NewFloat64(12.5))
		testSuccessful(t, "x - f", record, zed.NewFloat64(7.5))
		testSuccessful(t, "f - x", record, zed.NewFloat64(-7.5))
		testSuccessful(t, "x*f", record, zed.NewFloat64(25.0))
		testSuccessful(t, "f*x", record, zed.NewFloat64(25.0))
		testSuccessful(t, "x/f", record, zed.NewFloat64(4.0))
		testSuccessful(t, "f/x", record, zed.NewFloat64(0.25))
	}
	// Test string concatenation
	testSuccessful(t, `"hello" + " world"`, record, zed.NewString("hello world"))

	// Test string arithmetic other than + fails
	testSuccessful(t, `"hello" - " world"`, record, ZSON(`error("type string incompatible with '-' operator")`))
	testSuccessful(t, `"hello" * " world"`, record, ZSON(`error("type string incompatible with '*' operator")`))
	testSuccessful(t, `"hello" / " world"`, record, ZSON(`error("type string incompatible with '/' operator")`))

	// Test that addition fails on an unsupported type
	testSuccessful(t, "10.1.1.1 + 1", record, ZSON(`error("incompatible types")`))
	testSuccessful(t, "10.1.1.1 + 3.14159", record, ZSON(`error("incompatible types")`))
	testSuccessful(t, `10.1.1.1 + "foo"`, record, ZSON(`error("incompatible types")`))
}

func TestArrayIndex(t *testing.T) {
	const record = `{x:[1,2,3],i:1 (uint16)} (=0)`

	testSuccessful(t, "x[0]", record, zed.NewInt64(1))
	testSuccessful(t, "x[1]", record, zed.NewInt64(2))
	testSuccessful(t, "x[2]", record, zed.NewInt64(3))
	testSuccessful(t, "x[i]", record, zed.NewInt64(2))
	testSuccessful(t, "i+1", record, zed.NewInt64(2))
	testSuccessful(t, "x[i+1]", record, zed.NewInt64(3))
}

func TestFieldReference(t *testing.T) {
	const record = `{rec:{i:5 (int32),s:"boo",f:6.1} (=0)} (=1)`

	testSuccessful(t, "rec.i", record, zed.NewInt32(5))
	testSuccessful(t, "rec.s", record, zed.NewString("boo"))
	testSuccessful(t, "rec.f", record, zed.NewFloat64(6.1))
}

func TestConditional(t *testing.T) {
	const record = "{x:1}"

	testSuccessful(t, `x == 0 ? "zero" : "not zero"`, record, zed.NewString("not zero"))
	testSuccessful(t, `x == 1 ? "one" : "not one"`, record, zed.NewString("one"))
	testSuccessful(t, `x ? "x" : "not x"`, record, ZSON(`error({message:"?-operator: bool predicate required",on:1})`))

	// Ensure that the unevaluated clause doesn't generate errors
	// (field y doesn't exist but it shouldn't be evaluated)
	testSuccessful(t, "x == 0 ? y : x", record, zed.NewInt64(1))
	testSuccessful(t, "x != 0 ? x : y", record, zed.NewInt64(1))
}

func TestCasts(t *testing.T) {
	// Test casts to byte
	testSuccessful(t, "uint8(10)", "", zed.NewUint8(10))
	testSuccessful(t, "uint8(-1)", "", ZSON(`error({message:"cannot cast to uint8",on:-1})`))
	testSuccessful(t, "uint8(300)", "", ZSON(`error({message:"cannot cast to uint8",on:300})`))
	testSuccessful(t, `uint8("foo")`, "", ZSON(`error({message:"cannot cast to uint8",on:"foo"})`))

	// Test casts to int16
	testSuccessful(t, "int16(10)", "", ZSON(`10(int16)`))
	testSuccessful(t, "int16(-33000)", "", ZSON(`error({message:"cannot cast to int16",on:-33000})`))
	testSuccessful(t, "int16(33000)", "", ZSON(`error({message:"cannot cast to int16",on:33000})`))
	testSuccessful(t, `int16("foo")`, "", ZSON(`error({message:"cannot cast to int16",on:"foo"})`))

	// Test casts to uint16
	testSuccessful(t, "uint16(10)", "", zed.NewUint16(10))
	testSuccessful(t, "uint16(-1)", "", ZSON(`error({message:"cannot cast to uint16",on:-1})`))
	testSuccessful(t, "uint16(66000)", "", ZSON(`error({message:"cannot cast to uint16",on:66000})`))
	testSuccessful(t, `uint16("foo")`, "", ZSON(`error({message:"cannot cast to uint16",on:"foo"})`))

	// Test casts to int32
	testSuccessful(t, "int32(10)", "", zed.NewInt32(10))
	testSuccessful(t, "int32(-2200000000)", "", ZSON(`error({message:"cannot cast to int32",on:-2200000000})`))
	testSuccessful(t, "int32(2200000000)", "", ZSON(`error({message:"cannot cast to int32",on:2200000000})`))
	testSuccessful(t, `int32("foo")`, "", ZSON(`error({message:"cannot cast to int32",on:"foo"})`))

	// Test casts to uint32
	testSuccessful(t, "uint32(10)", "", zed.NewUint32(10))
	testSuccessful(t, "uint32(-1)", "", ZSON(`error({message:"cannot cast to uint32",on:-1})`))
	testSuccessful(t, "uint32(4300000000)", "", ZSON(`error({message:"cannot cast to uint32",on:4300000000})`))
	testSuccessful(t, `uint32("foo")`, "", ZSON(`error({message:"cannot cast to uint32",on:"foo"})`))

	// Test casts to uint64
	testSuccessful(t, "uint64(10)", "", zed.NewUint64(10))
	testSuccessful(t, "uint64(-1)", "", ZSON(`error({message:"cannot cast to uint64",on:-1})`))
	testSuccessful(t, `uint64("foo")`, "", ZSON(`error({message:"cannot cast to uint64",on:"foo"})`))

	// Test casts to float16
	testSuccessful(t, "float16(10)", "", zed.NewFloat16(10))
	testSuccessful(t, `float16("foo")`, "", ZSON(`error({message:"cannot cast to float16",on:"foo"})`))

	// Test casts to float32
	testSuccessful(t, "float32(10)", "", zed.NewFloat32(10))
	testSuccessful(t, `float32("foo")`, "", ZSON(`error({message:"cannot cast to float32",on:"foo"})`))

	// Test casts to float64
	testSuccessful(t, "float64(10)", "", zed.NewFloat64(10))
	testSuccessful(t, `float64("foo")`, "", ZSON(`error({message:"cannot cast to float64",on:"foo"})`))

	// Test casts to ip
	testSuccessful(t, `ip("1.2.3.4")`, "", zed.NewIP(netip.MustParseAddr("1.2.3.4")))
	testSuccessful(t, "ip(1234)", "", ZSON(`error({message:"cannot cast to ip",on:1234})`))
	testSuccessful(t, `ip("not an address")`, "", ZSON(`error({message:"cannot cast to ip",on:"not an address"})`))

	// Test casts to net
	testSuccessful(t, `net("1.2.3.0/24")`, "", zed.NewNet(netip.MustParsePrefix("1.2.3.0/24")))
	testSuccessful(t, "net(1234)", "", ZSON(`error({message:"cannot cast to net",on:1234})`))
	testSuccessful(t, `net("not an address")`, "", ZSON(`error({message:"cannot cast to net",on:"not an address"})`))
	testSuccessful(t, `net(1.2.3.4)`, "", ZSON(`error({message:"cannot cast to net",on:1.2.3.4})`))

	// Test casts to time
	const ts = 1589126400_000_000_000
	// float16 lacks sufficient precision to represent ts exactly.
	testSuccessful(t, "time(float16(1589126400000000000))", "", zed.NewTime(nano.Ts(float16.Fromfloat32(float32(ts)).Float32())))
	// float32 lacks sufficient precision to represent ts exactly.
	testSuccessful(t, "time(float32(1589126400000000000))", "", zed.NewTime(nano.Ts(float32(ts))))
	testSuccessful(t, "time(float64(1589126400000000000))", "", zed.NewTime(ts))
	testSuccessful(t, "time(1589126400000000000)", "", zed.NewTime(ts))
	testSuccessful(t, `time("1589126400000000000")`, "", zed.NewTime(ts))

	testSuccessful(t, "string(1.2)", "", zed.NewString("1.2"))
	testSuccessful(t, "string(5)", "", zed.NewString("5"))
	testSuccessful(t, "string(1.2.3.4)", "", zed.NewString("1.2.3.4"))
	testSuccessful(t, `int64("1")`, "", zed.NewInt64(1))
	testSuccessful(t, `int64("-1")`, "", zed.NewInt64(-1))
	testSuccessful(t, `float16("5.5")`, "", zed.NewFloat16(5.5))
	testSuccessful(t, `float32("5.5")`, "", zed.NewFloat32(5.5))
	testSuccessful(t, `float64("5.5")`, "", zed.NewFloat64(5.5))
	testSuccessful(t, `ip("1.2.3.4")`, "", zed.NewIP(netip.MustParseAddr("1.2.3.4")))

	testSuccessful(t, "ip(1)", "", ZSON(`error({message:"cannot cast to ip",on:1})`))
	testSuccessful(t, `int64("abc")`, "", ZSON(`error({message:"cannot cast to int64",on:"abc"})`))
	testSuccessful(t, `float16("abc")`, "", ZSON(`error({message:"cannot cast to float16",on:"abc"})`))
	testSuccessful(t, `float32("abc")`, "", ZSON(`error({message:"cannot cast to float32",on:"abc"})`))
	testSuccessful(t, `float64("abc")`, "", ZSON(`error({message:"cannot cast to float64",on:"abc"})`))
	testSuccessful(t, `ip("abc")`, "", ZSON(`error({message:"cannot cast to ip",on:"abc"})`))
}
