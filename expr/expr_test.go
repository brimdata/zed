package expr_test

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/ztest"
	"github.com/stretchr/testify/require"
)

func testSuccessful(t *testing.T, e string, input string, expectedVal zed.Value) {
	if input == "" {
		input = "{}"
	}
	expected := zson.MustFormatValue(expectedVal)
	runZTest(t, e, &ztest.ZTest{
		Zed:    fmt.Sprintf("yield %s", e),
		Input:  input,
		Output: expected + "\n",
	})
}

func testError(t *testing.T, e string, expectErr error, description string) {
	runZTest(t, e, &ztest.ZTest{
		Zed:     fmt.Sprintf("yield %s", e),
		ErrorRE: expectErr.Error(),
	})
}

func runZTest(t *testing.T, e string, zt *ztest.ZTest) {
	t.Run(e, func(t *testing.T) {
		t.Parallel()
		if err := zt.RunInternal(""); err != nil {
			t.Fatal(err)
		}
	})
}

func zbool(b bool) zed.Value {
	return zed.Value{zed.TypeBool, zed.EncodeBool(b)}
}

func zerr(s string) zed.Value {
	return zed.Value{zed.TypeError, zed.EncodeString(s)}
}

func zint32(v int32) zed.Value {
	return zed.Value{zed.TypeInt32, zed.EncodeInt(int64(v))}
}

func zint64(v int64) zed.Value {
	return zed.Value{zed.TypeInt64, zed.EncodeInt(v)}
}

func zuint64(v uint64) zed.Value {
	return zed.Value{zed.TypeUint64, zed.EncodeUint(v)}
}

func zfloat32(f float32) zed.Value {
	return zed.Value{zed.TypeFloat32, zed.EncodeFloat32(f)}
}

func zfloat64(f float64) zed.Value {
	return zed.Value{zed.TypeFloat64, zed.EncodeFloat64(f)}
}

func zstring(s string) zed.Value {
	return zed.Value{zed.TypeString, zed.EncodeString(s)}
}

func zip(t *testing.T, s string) zed.Value {
	ip := net.ParseIP(s)
	require.NotNil(t, ip, "converted ip")
	return zed.Value{zed.TypeIP, zed.EncodeIP(ip)}
}
func znet(t *testing.T, s string) zed.Value {
	_, net, err := net.ParseCIDR(s)
	require.NoError(t, err)
	return zed.Value{zed.TypeNet, zed.EncodeNet(net)}
}

func TestPrimitives(t *testing.T) {
	const record = `{x:10 (int32),f:2.5,s:"hello"} (=0)`

	// Test simple literals
	testSuccessful(t, "50", record, zint64(50))
	testSuccessful(t, "3.14", record, zfloat64(3.14))
	testSuccessful(t, `"boo"`, record, zstring("boo"))

	// Test good field references
	testSuccessful(t, "x", record, zint32(10))
	testSuccessful(t, "f", record, zfloat64(2.5))
	testSuccessful(t, "s", record, zstring("hello"))
}

func TestCompareNumbers(t *testing.T) {
	var numericTypes = []string{
		"uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64"}
	var intFields = []string{"u8", "i16", "u16", "i32", "u32", "i64", "u64"}

	for _, typ := range numericTypes {
		// Make a test point with this type in a field called x plus
		// one field of each other integer type
		one := "1"
		if strings.HasPrefix(typ, "float") {
			one = "1."
		}
		record := fmt.Sprintf(
			"{x:%s (%s),u8:0 (uint8),i16:0 (int16),u16:0 (uint16),i32:0 (int32),u32:0 (uint32),i64:0,u64:0 (uint64)} (=0)",
			one, typ)
		// Test the 6 comparison operators against a constant
		testSuccessful(t, "x == 1", record, zbool(true))
		testSuccessful(t, "x == 0", record, zbool(false))
		testSuccessful(t, "x != 0", record, zbool(true))
		testSuccessful(t, "x != 1", record, zbool(false))
		testSuccessful(t, "x < 2", record, zbool(true))
		testSuccessful(t, "x < 1", record, zbool(false))
		testSuccessful(t, "x <= 2", record, zbool(true))
		testSuccessful(t, "x <= 1", record, zbool(true))
		testSuccessful(t, "x <= 0", record, zbool(false))
		testSuccessful(t, "x > 0", record, zbool(true))
		testSuccessful(t, "x > 1", record, zbool(false))
		testSuccessful(t, "x >= 0", record, zbool(true))
		testSuccessful(t, "x >= 1", record, zbool(true))
		testSuccessful(t, "x >= 2", record, zbool(false))

		// Test the full matrix of comparisons between all
		// the integer types
		for _, other := range intFields {
			exp := fmt.Sprintf("x == %s", other)
			testSuccessful(t, exp, record, zbool(false))

			exp = fmt.Sprintf("x != %s", other)
			testSuccessful(t, exp, record, zbool(true))

			exp = fmt.Sprintf("x < %s", other)
			testSuccessful(t, exp, record, zbool(false))

			exp = fmt.Sprintf("x <= %s", other)
			testSuccessful(t, exp, record, zbool(false))

			exp = fmt.Sprintf("x > %s", other)
			testSuccessful(t, exp, record, zbool(true))

			exp = fmt.Sprintf("x >= %s", other)
			testSuccessful(t, exp, record, zbool(true))
		}

		// For integer types, test this type against other
		// number-ish types: port, time, duration
		if !strings.HasPrefix(typ, "float") {
			record := fmt.Sprintf(
				"{x:%s (%s),p:80 (port=(uint16)),t:2020-03-09T22:54:12Z,d:16m40s} (=0)", one, typ)

			// port
			testSuccessful(t, "x == p", record, zbool(false))
			testSuccessful(t, "p == x", record, zbool(false))
			testSuccessful(t, "x != p", record, zbool(true))
			testSuccessful(t, "p != x", record, zbool(true))
			testSuccessful(t, "x < p", record, zbool(true))
			testSuccessful(t, "p < x", record, zbool(false))
			testSuccessful(t, "x <= p", record, zbool(true))
			testSuccessful(t, "p <= x", record, zbool(false))
			testSuccessful(t, "x > p", record, zbool(false))
			testSuccessful(t, "p > x", record, zbool(true))
			testSuccessful(t, "x >= p", record, zbool(false))
			testSuccessful(t, "p >= x", record, zbool(true))

			// time
			testSuccessful(t, "x == t", record, zbool(false))
			testSuccessful(t, "t == x", record, zbool(false))
			testSuccessful(t, "x != t", record, zbool(true))
			testSuccessful(t, "t != x", record, zbool(true))
			testSuccessful(t, "x < t", record, zbool(true))
			testSuccessful(t, "t < x", record, zbool(false))
			testSuccessful(t, "x <= t", record, zbool(true))
			testSuccessful(t, "t <= x", record, zbool(false))
			testSuccessful(t, "x > t", record, zbool(false))
			testSuccessful(t, "t > x", record, zbool(true))
			testSuccessful(t, "x >= t", record, zbool(false))
			testSuccessful(t, "t >= x", record, zbool(true))

			// duration
			testSuccessful(t, "x == d", record, zbool(false))
			testSuccessful(t, "d == x", record, zbool(false))
			testSuccessful(t, "x != d", record, zbool(true))
			testSuccessful(t, "d != x", record, zbool(true))
			testSuccessful(t, "x < d", record, zbool(true))
			testSuccessful(t, "d < x", record, zbool(false))
			testSuccessful(t, "x <= d", record, zbool(true))
			testSuccessful(t, "d <= x", record, zbool(false))
			testSuccessful(t, "x > d", record, zbool(false))
			testSuccessful(t, "d > x", record, zbool(true))
			testSuccessful(t, "x >= d", record, zbool(false))
			testSuccessful(t, "d >= x", record, zbool(true))
		}

		// Test this type against non-numeric types
		record = fmt.Sprintf(
			`{x:%s (%s),s:"hello",i:10.1.1.1,n:10.1.0.0/16} (=0)`, one, typ)

		testSuccessful(t, "x == s", record, ZSON("false"))
		testSuccessful(t, "x != s", record, ZSON("false"))
		testSuccessful(t, "x < s", record, ZSON("false"))
		testSuccessful(t, "x <= s", record, ZSON("false"))
		testSuccessful(t, "x > s", record, ZSON("false"))
		testSuccessful(t, "x >= s", record, ZSON("false"))

		testSuccessful(t, "x == i", record, ZSON("false"))
		testSuccessful(t, "x != i", record, ZSON("false"))
		testSuccessful(t, "x < i", record, ZSON("false"))
		testSuccessful(t, "x <= i", record, ZSON("false"))
		testSuccessful(t, "x > i", record, ZSON("false"))
		testSuccessful(t, "x >= i", record, ZSON("false"))

		testSuccessful(t, "x == n", record, ZSON("false"))
		testSuccessful(t, "x != n", record, ZSON("false"))
		testSuccessful(t, "x < n", record, ZSON("false"))
		testSuccessful(t, "x <= n", record, ZSON("false"))
		testSuccessful(t, "x > n", record, ZSON("false"))
		testSuccessful(t, "x >= n", record, ZSON("false"))
	}

	// Test comparison between signed and unsigned and also
	// floats that cast to different integers.
	const rec2 = "{i:-1,u:18446744073709551615 (uint64),f:-1.} (=0)"

	testSuccessful(t, "i == u", rec2, zbool(false))
	testSuccessful(t, "i != u", rec2, zbool(true))
	testSuccessful(t, "i < u", rec2, zbool(true))
	testSuccessful(t, "i <= u", rec2, zbool(true))
	testSuccessful(t, "i > u", rec2, zbool(false))
	testSuccessful(t, "i >= u", rec2, zbool(false))

	testSuccessful(t, "u == i", rec2, zbool(false))
	testSuccessful(t, "u != i", rec2, zbool(true))
	testSuccessful(t, "u < i", rec2, zbool(false))
	testSuccessful(t, "u <= i", rec2, zbool(false))
	testSuccessful(t, "u > i", rec2, zbool(true))
	testSuccessful(t, "u >= i", rec2, zbool(true))

	testSuccessful(t, "f == u", rec2, zbool(false))
	testSuccessful(t, "f != u", rec2, zbool(true))
	testSuccessful(t, "f < u", rec2, zbool(true))
	testSuccessful(t, "f <= u", rec2, zbool(true))
	testSuccessful(t, "f > u", rec2, zbool(false))
	testSuccessful(t, "f >= u", rec2, zbool(false))

	testSuccessful(t, "u == f", rec2, zbool(false))
	testSuccessful(t, "u != f", rec2, zbool(true))
	testSuccessful(t, "u < f", rec2, zbool(false))
	testSuccessful(t, "u <= f", rec2, zbool(false))
	testSuccessful(t, "u > f", rec2, zbool(true))
	testSuccessful(t, "u >= f", rec2, zbool(true))
}

func TestCompareNonNumbers(t *testing.T) {
	record := `
{
    b: true,
    s: "hello",
    i: 10.1.1.1,
    p: 443 (port=(uint16)),
    net: 10.1.0.0/16,
    t: 2020-03-09T22:54:12Z,
    d: 16m40s
} (=0)
`

	// bool
	testSuccessful(t, "b == true", record, zbool(true))
	testSuccessful(t, "b == false", record, zbool(false))
	testSuccessful(t, "b != true", record, zbool(false))
	testSuccessful(t, "b != false", record, zbool(true))

	// string
	testSuccessful(t, `s == "hello"`, record, zbool(true))
	testSuccessful(t, `s != "hello"`, record, zbool(false))
	testSuccessful(t, `s == "world"`, record, zbool(false))
	testSuccessful(t, `s != "world"`, record, zbool(true))

	// ip
	testSuccessful(t, "i == 10.1.1.1", record, zbool(true))
	testSuccessful(t, "i != 10.1.1.1", record, zbool(false))
	testSuccessful(t, "i == 1.1.1.10", record, zbool(false))
	testSuccessful(t, "i != 1.1.1.10", record, zbool(true))
	testSuccessful(t, "i == i", record, zbool(true))

	// port
	testSuccessful(t, "p == 443", record, zbool(true))
	testSuccessful(t, "p != 443", record, zbool(false))

	// net
	testSuccessful(t, "net == 10.1.0.0/16", record, zbool(true))
	testSuccessful(t, "net != 10.1.0.0/16", record, zbool(false))
	testSuccessful(t, "net == 10.1.0.0/24", record, zbool(false))
	testSuccessful(t, "net != 10.1.0.0/24", record, zbool(true))

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
				testSuccessful(t, exp, record, ZSON(`false`))
			}
		}
	}

	// relative comparisons on strings
	record = `{s:"abc"}`

	testSuccessful(t, `s < "brim"`, record, zbool(true))
	testSuccessful(t, `s < "aaa"`, record, zbool(false))
	testSuccessful(t, `s < "abc"`, record, zbool(false))

	testSuccessful(t, `s > "brim"`, record, zbool(false))
	testSuccessful(t, `s > "aaa"`, record, zbool(true))
	testSuccessful(t, `s > "abc"`, record, zbool(false))

	testSuccessful(t, `s <= "brim"`, record, zbool(true))
	testSuccessful(t, `s <= "aaa"`, record, zbool(false))
	testSuccessful(t, `s <= "abc"`, record, zbool(true))

	testSuccessful(t, `s >= "brim"`, record, zbool(false))
	testSuccessful(t, `s >= "aaa"`, record, zbool(true))
	testSuccessful(t, `s >= "abc"`, record, zbool(true))
}

func TestPattern(t *testing.T) {
	testSuccessful(t, `"abc" == "abc"`, "", zbool(true))
	testSuccessful(t, `"abc" != "abc"`, "", zbool(false))
	testSuccessful(t, "10.1.1.1 in 10.0.0.0/8", "", zbool(true))
	testSuccessful(t, "10.1.1.1 in 192.168.0.0/16", "", zbool(false))
	testSuccessful(t, "!(10.1.1.1 in 10.0.0.0/8)", "", zbool(false))
	testSuccessful(t, "!(10.1.1.1 in 192.168.0.0/16)", "", zbool(true))
}

func TestIn(t *testing.T) {
	const record = "{a:[1 (int32),2 (int32),3 (int32)] (=0),s:|[4 (int32),5 (int32),6 (int32)]| (=1)} (=2)"

	testSuccessful(t, "1 in a", record, zbool(true))
	testSuccessful(t, "0 in a", record, zbool(false))

	testSuccessful(t, "1 in s", record, zbool(false))
	testSuccessful(t, "4 in s", record, zbool(true))

	testSuccessful(t, `"boo" in a`, record, zbool(false))
	testSuccessful(t, `"boo" in s`, record, zbool(false))
	testSuccessful(t, "1 in 2", record, zerr("'in' operator applied to non-container type"))
}

func TestArithmetic(t *testing.T) {
	record := "{x:10 (int32),f:2.5} (=0)"

	// Test integer arithmetic
	testSuccessful(t, "100 + 23", record, zint64(123))
	testSuccessful(t, "x + 5", record, zint64(15))
	testSuccessful(t, "5 + x", record, zint64(15))
	testSuccessful(t, "x - 5", record, zint64(5))
	testSuccessful(t, "0 - x", record, zint64(-10))
	testSuccessful(t, "x + 5 - 3", record, zint64(12))
	testSuccessful(t, "x*2", record, zint64(20))
	testSuccessful(t, "5*x*2", record, zint64(100))
	testSuccessful(t, "x/3", record, zint64(3))
	testSuccessful(t, "20/x", record, zint64(2))

	// Test precedence of arithmetic operations
	testSuccessful(t, "x + 1 * 10", record, zint64(20))
	testSuccessful(t, "(x + 1) * 10", record, zint64(110))

	// Test arithmetic with floats
	testSuccessful(t, "f + 1.0", record, zfloat64(3.5))
	testSuccessful(t, "1.0 + f", record, zfloat64(3.5))
	testSuccessful(t, "f - 1.0", record, zfloat64(1.5))
	testSuccessful(t, "0.0 - f", record, zfloat64(-2.5))
	testSuccessful(t, "f * 1.5", record, zfloat64(3.75))
	testSuccessful(t, "1.5 * f", record, zfloat64(3.75))
	testSuccessful(t, "f / 1.25", record, zfloat64(2.0))
	testSuccessful(t, "5.0 / f", record, zfloat64(2.0))

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
			return zed.Value{signed(w), zed.AppendInt(nil, int64(v))}
		}
		return zed.Value{unsigned(w), zed.AppendUint(nil, v)}
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
		testSuccessful(t, "f + 5", record, zfloat64(7.5))
		testSuccessful(t, "5 + f", record, zfloat64(7.5))
		testSuccessful(t, "f + x", record, zfloat64(12.5))
		testSuccessful(t, "x + f", record, zfloat64(12.5))
		testSuccessful(t, "x - f", record, zfloat64(7.5))
		testSuccessful(t, "f - x", record, zfloat64(-7.5))
		testSuccessful(t, "x*f", record, zfloat64(25.0))
		testSuccessful(t, "f*x", record, zfloat64(25.0))
		testSuccessful(t, "x/f", record, zfloat64(4.0))
		testSuccessful(t, "f/x", record, zfloat64(0.25))
	}
	// Test string concatenation
	testSuccessful(t, `"hello" + " world"`, record, zstring("hello world"))

	// Test string arithmetic other than + fails
	testSuccessful(t, `"hello" - " world"`, record, ZSON(`"type string incompatible with '-' operator"(error)`))
	testSuccessful(t, `"hello" * " world"`, record, ZSON(`"type string incompatible with '*' operator"(error)`))
	testSuccessful(t, `"hello" / " world"`, record, ZSON(`"type string incompatible with '/' operator"(error)`))

	// Test that addition fails on an unsupported type
	testSuccessful(t, "10.1.1.1 + 1", record, ZSON(`"incompatible types"(error)`))
	testSuccessful(t, "10.1.1.1 + 3.14159", record, ZSON(`"incompatible types"(error)`))
	testSuccessful(t, `10.1.1.1 + "foo"`, record, ZSON(`"incompatible types"(error)`))
}

func TestArrayIndex(t *testing.T) {
	const record = `{x:[1,2,3],i:1 (uint16)} (=0)`

	testSuccessful(t, "x[0]", record, zint64(1))
	testSuccessful(t, "x[1]", record, zint64(2))
	testSuccessful(t, "x[2]", record, zint64(3))
	testSuccessful(t, "x[i]", record, zint64(2))
	testSuccessful(t, "i+1", record, zint64(2))
	testSuccessful(t, "x[i+1]", record, zint64(3))
}

func TestFieldReference(t *testing.T) {
	const record = `{rec:{i:5 (int32),s:"boo",f:6.1} (=0)} (=1)`

	testSuccessful(t, "rec.i", record, zint32(5))
	testSuccessful(t, "rec.s", record, zstring("boo"))
	testSuccessful(t, "rec.f", record, zfloat64(6.1))
}

func TestConditional(t *testing.T) {
	const record = "{x:1}"

	testSuccessful(t, `x == 0 ? "zero" : "not zero"`, record, zstring("not zero"))
	testSuccessful(t, `x == 1 ? "one" : "not one"`, record, zstring("one"))
	testSuccessful(t, `x ? "x" : "not x"`, record, ZSON(`"?-operator: bool predicate required"(error)`))

	// Ensure that the unevaluated clause doesn't generate errors
	// (field y doesn't exist but it shouldn't be evaluated)
	testSuccessful(t, "x == 0 ? y : x", record, zint64(1))
	testSuccessful(t, "x != 0 ? x : y", record, zint64(1))
}

func TestCasts(t *testing.T) {
	// Test casts to byte
	testSuccessful(t, "uint8(10)", "", zed.Value{zed.TypeUint8, zed.EncodeUint(10)})
	testSuccessful(t, "uint8(-1)", "", ZSON(`"cannot cast -1 to type uint8"(error)`))
	testSuccessful(t, "uint8(300)", "", ZSON(`"cannot cast 300 to type uint8"(error)`))
	testSuccessful(t, `uint8("foo")`, "", ZSON(`"cannot cast \"foo\" to type uint8"(error)`))

	// Test casts to int16
	testSuccessful(t, "int16(10)", "", ZSON(`10(int16)`))
	testSuccessful(t, "int16(-33000)", "", ZSON(`"cannot cast -33000 to type int16"(error)`))
	testSuccessful(t, "int16(33000)", "", ZSON(`"cannot cast 33000 to type int16"(error)`))
	testSuccessful(t, `int16("foo")`, "", ZSON(`"cannot cast \"foo\" to type int16"(error)`))

	// Test casts to uint16
	testSuccessful(t, "uint16(10)", "", zed.Value{zed.TypeUint16, zed.EncodeUint(10)})
	testSuccessful(t, "uint16(-1)", "", ZSON(`"cannot cast -1 to type uint16"(error)`))
	testSuccessful(t, "uint16(66000)", "", ZSON(`"cannot cast 66000 to type uint16"(error)`))
	testSuccessful(t, `uint16("foo")`, "", ZSON(`"cannot cast \"foo\" to type uint16"(error)`))

	// Test casts to int32
	testSuccessful(t, "int32(10)", "", zed.Value{zed.TypeInt32, zed.EncodeInt(10)})
	testSuccessful(t, "int32(-2200000000)", "", ZSON(`"cannot cast -2200000000 to type int32"(error)`))
	testSuccessful(t, "int32(2200000000)", "", ZSON(`"cannot cast 2200000000 to type int32"(error)`))
	testSuccessful(t, `int32("foo")`, "", ZSON(`"cannot cast \"foo\" to type int32"(error)`))

	// Test casts to uint32
	testSuccessful(t, "uint32(10)", "", zed.Value{zed.TypeUint32, zed.EncodeUint(10)})
	testSuccessful(t, "uint32(-1)", "", ZSON(`"cannot cast -1 to type uint32"(error)`))
	testSuccessful(t, "uint32(4300000000)", "", ZSON(`"cannot cast 4300000000 to type uint32"(error)`))
	testSuccessful(t, `uint32("foo")`, "", ZSON(`"cannot cast \"foo\" to type uint32"(error)`))

	// Test casts to uint64
	testSuccessful(t, "uint64(10)", "", zuint64(10))
	testSuccessful(t, "uint64(-1)", "", ZSON(`"cannot cast -1 to type uint64"(error)`))
	testSuccessful(t, `uint64("foo")`, "", ZSON(`"cannot cast \"foo\" to type uint64"(error)`))

	// Test casts to float32
	testSuccessful(t, "float32(10)", "", zfloat32(10))
	testSuccessful(t, `float32("foo")`, "", ZSON(`"cannot cast \"foo\" to type float32"(error)`))

	// Test casts to float64
	testSuccessful(t, "float64(10)", "", zfloat64(10))
	testSuccessful(t, `float64("foo")`, "", ZSON(`"cannot cast \"foo\" to type float64"(error)`))

	// Test casts to ip
	testSuccessful(t, `ip("1.2.3.4")`, "", zip(t, "1.2.3.4"))
	testSuccessful(t, "ip(1234)", "", ZSON(`"cannot cast 1234 to type ip"(error)`))
	testSuccessful(t, `ip("not an address")`, "", ZSON(`"cannot cast \"not an address\" to type ip"(error)`))

	// Test casts to net
	testSuccessful(t, `net("1.2.3.0/24")`, "", znet(t, "1.2.3.0/24"))
	testSuccessful(t, "net(1234)", "", ZSON(`"cannot cast 1234 to type net"(error)`))
	testSuccessful(t, `net("not an address")`, "", ZSON(`"cannot cast \"not an address\" to type net"(error)`))
	testSuccessful(t, `net(1.2.3.4)`, "", ZSON(`"cannot cast 1.2.3.4 to type net"(error)`))

	// Test casts to time
	ts := zed.Value{zed.TypeTime, zed.EncodeTime(nano.Ts(1589126400_000_000_000))}
	testSuccessful(t, "time(float32(1589126400.0))", "", ts)
	testSuccessful(t, "time(float64(1589126400.0))", "", ts)
	testSuccessful(t, "time(1589126400000000000)", "", ts)
	testSuccessful(t, `time("1589126400")`, "", ts)

	testSuccessful(t, "string(1.2)", "", zstring("1.2"))
	testSuccessful(t, "string(5)", "", zstring("5"))
	testSuccessful(t, "string(1.2.3.4)", "", zstring("1.2.3.4"))
	testSuccessful(t, `int64("1")`, "", zint64(1))
	testSuccessful(t, `int64("-1")`, "", zint64(-1))
	testSuccessful(t, `float32("5.5")`, "", zfloat32(5.5))
	testSuccessful(t, `float64("5.5")`, "", zfloat64(5.5))
	testSuccessful(t, `ip("1.2.3.4")`, "", zaddr("1.2.3.4"))

	testSuccessful(t, "ip(1)", "", ZSON(`"cannot cast 1 to type ip"(error)`))
	testSuccessful(t, `int64("abc")`, "", ZSON(`"cannot cast \"abc\" to type int64"(error)`))
	testSuccessful(t, `float32("abc")`, "", ZSON(`"cannot cast \"abc\" to type float32"(error)`))
	testSuccessful(t, `float64("abc")`, "", ZSON(`"cannot cast \"abc\" to type float64"(error)`))
	testSuccessful(t, `ip("abc")`, "", ZSON(`"cannot cast \"abc\" to type ip"(error)`))
}
