package expr_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// XXX copied from filter_test.go where could we put a single copy of this?
func parseOneRecord(zngsrc string) (*zng.Record, error) {
	ior := strings.NewReader(zngsrc)
	reader, err := detector.LookupReader("zng", ior, resolver.NewContext())
	if err != nil {
		return nil, err
	}

	rec, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.New("expected to read one record")
	}
	rec.Keep()

	rec2, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if rec2 != nil {
		return nil, errors.New("got more than one record")
	}
	return rec, nil
}

func compileExpr(s string) (expr.ExpressionEvaluator, error) {
	parsed, err := zql.Parse("", []byte(s), zql.Entrypoint("Expression"))
	if err != nil {
		return nil, err
	}

	node, ok := parsed.(ast.Expression)
	if !ok {
		return nil, errors.New("expected Expression")
	}

	return expr.CompileExpr(node)
}

// Compile and evaluate a zql expression against a provided Record.
// Returns the resulting Value if successful or an error otherwise
// (which could be failure to compile the expression or failure while
// evaluating the expression).
func evaluate(e string, record *zng.Record) (zng.Value, error) {
	eval, err := compileExpr(e)
	if err != nil {
		return zng.Value{}, err
	}

	// And execute it.
	return eval(record)
}

func testSuccessful(t *testing.T, e string, record *zng.Record, expect zng.Value) {
	t.Run(e, func(t *testing.T) {
		result, err := evaluate(e, record)
		require.NoError(t, err)

		assert.Equal(t, expect.Type, result.Type, "result type is correct")
		assert.Equal(t, expect.Bytes, result.Bytes, "result value is correct")
	})
}

func testError(t *testing.T, e string, record *zng.Record, expectErr error, description string) {
	t.Run(description, func(t *testing.T) {
		_, err := evaluate(e, record)
		assert.Errorf(t, err, "got error when %s", description)
		assert.True(t, errors.Is(err, expectErr), "got correct error when %s", description)
	})
}

func zbool(b bool) zng.Value {
	return zng.Value{zng.TypeBool, zng.EncodeBool(b)}
}

func zint32(v int32) zng.Value {
	return zng.Value{zng.TypeInt32, zng.EncodeInt(int64(v))}
}

func zint64(v int64) zng.Value {
	return zng.Value{zng.TypeInt64, zng.EncodeInt(v)}
}

func zuint64(v uint64) zng.Value {
	return zng.Value{zng.TypeUint64, zng.EncodeUint(v)}
}

func zfloat64(f float64) zng.Value {
	return zng.Value{zng.TypeFloat64, zng.EncodeFloat64(f)}
}

func zstring(s string) zng.Value {
	return zng.Value{zng.TypeString, zng.EncodeString(s)}
}

func TestPrimitives(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[x:int32,f:float64,s:string]
0:[10;2.5;hello;]`)
	require.NoError(t, err)

	// Test simple literals
	testSuccessful(t, "50", record, zint64(50))
	testSuccessful(t, "3.14", record, zfloat64(3.14))
	testSuccessful(t, `"boo"`, record, zstring("boo"))

	// Test good field references
	testSuccessful(t, "x", record, zint32(10))
	testSuccessful(t, "f", record, zfloat64(2.5))
	testSuccessful(t, "s", record, zstring("hello"))

	// Test bad field reference
	testError(t, "doesnexist", record, expr.ErrNoSuchField, "referencing non-existent field")
}

func TestComplex(t *testing.T) {
	// Test that an expression can evaluate to a complex type
	record, err := parseOneRecord(`
#0:record[r:record[s:string]]
0:[[hello;]]`)
	require.NoError(t, err)
	result, err := evaluate("r", record)
	require.NoError(t, err)
	recType, ok := result.Type.(*zng.TypeRecord)
	assert.True(t, ok, "result type is record")
	assert.Equal(t, 1, len(recType.Columns), "result has one column")
	assert.Equal(t, zng.TypeString, recType.Columns[0].Type, "result has string column")
}

func TestLogical(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[t:bool,f:bool]
0:[T;F;]`)
	require.NoError(t, err)

	testSuccessful(t, "t AND t", record, zbool(true))
	testSuccessful(t, "t AND f", record, zbool(false))
	testSuccessful(t, "f AND t", record, zbool(false))
	testSuccessful(t, "f AND f", record, zbool(false))

	testSuccessful(t, "t OR t", record, zbool(true))
	testSuccessful(t, "t OR f", record, zbool(true))
	testSuccessful(t, "f OR t", record, zbool(true))
	testSuccessful(t, "f OR f", record, zbool(false))

	testSuccessful(t, "!t", record, zbool(false))
	testSuccessful(t, "!f", record, zbool(true))
	testSuccessful(t, "!!f", record, zbool(false))
}

func TestCompareNumbers(t *testing.T) {
	var numericTypes = []string{"byte", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float64"}
	var intFields = []string{"u8", "i16", "u16", "i32", "u32", "i64", "u64"}

	for _, typ := range numericTypes {
		// Make a test point with this type in a field called x plus
		// one field of each other integer type
		src := fmt.Sprintf(`
#0:record[x:%s,u8:byte,i16:int16,u16:uint16,i32:int32,u32:uint32,i64:int64,u64:uint16]
0:[1;0;0;0;0;0;0;0;]`, typ)
		record, err := parseOneRecord(src)
		require.NoError(t, err)

		// Test the 6 comparison operators against a constant
		testSuccessful(t, "x = 1", record, zbool(true))
		testSuccessful(t, "x = 0", record, zbool(false))
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
			exp := fmt.Sprintf("x = %s", other)
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
		if typ != "float64" {
			src = fmt.Sprintf(`
#0:record[x:%s,p:port,t:time,d:duration]
0:[1;80;1583794452;1000;]`, typ)
			record, err = parseOneRecord(src)
			require.NoError(t, err)

			// port
			testSuccessful(t, "x = p", record, zbool(false))
			testSuccessful(t, "p = x", record, zbool(false))
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
			testSuccessful(t, "x = t", record, zbool(false))
			testSuccessful(t, "t = x", record, zbool(false))
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
			testSuccessful(t, "x = d", record, zbool(false))
			testSuccessful(t, "d = x", record, zbool(false))
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
		src = fmt.Sprintf(`
#0:record[x:%s,s:string,bs:bstring,i:ip,n:net]
0:[1;hello;world;10.1.1.1;10.1.0.0/16;]`, typ)
		record, err = parseOneRecord(src)
		require.NoError(t, err)

		testError(t, "x = s", record, expr.ErrIncompatibleTypes, "comparing integer and string")
		testError(t, "x != s", record, expr.ErrIncompatibleTypes, "comparing integer and string")
		testError(t, "x < s", record, expr.ErrIncompatibleTypes, "comparing integer and string")
		testError(t, "x <= s", record, expr.ErrIncompatibleTypes, "comparing integer and string")
		testError(t, "x > s", record, expr.ErrIncompatibleTypes, "comparing integer and string")
		testError(t, "x >= s", record, expr.ErrIncompatibleTypes, "comparing integer and string")

		testError(t, "x = bs", record, expr.ErrIncompatibleTypes, "comparing integer and bstring")
		testError(t, "x != bs", record, expr.ErrIncompatibleTypes, "comparing integer and bstring")
		testError(t, "x < bs", record, expr.ErrIncompatibleTypes, "comparing integer and bstring")
		testError(t, "x <= bs", record, expr.ErrIncompatibleTypes, "comparing integer and bstring")
		testError(t, "x > bs", record, expr.ErrIncompatibleTypes, "comparing integer and bstring")
		testError(t, "x >= bs", record, expr.ErrIncompatibleTypes, "comparing integer and bstring")

		testError(t, "x = i", record, expr.ErrIncompatibleTypes, "comparing integer and ip")
		testError(t, "x != i", record, expr.ErrIncompatibleTypes, "comparing integer and ip")
		testError(t, "x < i", record, expr.ErrIncompatibleTypes, "comparing integer and ip")
		testError(t, "x <= i", record, expr.ErrIncompatibleTypes, "comparing integer and ip")
		testError(t, "x > i", record, expr.ErrIncompatibleTypes, "comparing integer and ip")
		testError(t, "x >= i", record, expr.ErrIncompatibleTypes, "comparing integer and ip")

		testError(t, "x = n", record, expr.ErrIncompatibleTypes, "comparing integer and net")
		testError(t, "x != n", record, expr.ErrIncompatibleTypes, "comparing integer and net")
		testError(t, "x < n", record, expr.ErrIncompatibleTypes, "comparing integer and net")
		testError(t, "x <= n", record, expr.ErrIncompatibleTypes, "comparing integer and net")
		testError(t, "x > n", record, expr.ErrIncompatibleTypes, "comparing integer and net")
		testError(t, "x >= n", record, expr.ErrIncompatibleTypes, "comparing integer and string")
	}

	// Test comparison between signed and unsigned and also
	// floats that cast to different integers.
	rec2, err := parseOneRecord(`
#0:record[i:int64,u:uint64,f:float64]
0:[-1;18446744073709551615;-1.0;]`)
	require.NoError(t, err)
	testSuccessful(t, "i = u", rec2, zbool(false))
	testSuccessful(t, "i != u", rec2, zbool(true))
	testSuccessful(t, "i < u", rec2, zbool(true))
	testSuccessful(t, "i <= u", rec2, zbool(true))
	testSuccessful(t, "i > u", rec2, zbool(false))
	testSuccessful(t, "i >= u", rec2, zbool(false))

	testSuccessful(t, "u = i", rec2, zbool(false))
	testSuccessful(t, "u != i", rec2, zbool(true))
	testSuccessful(t, "u < i", rec2, zbool(false))
	testSuccessful(t, "u <= i", rec2, zbool(false))
	testSuccessful(t, "u > i", rec2, zbool(true))
	testSuccessful(t, "u >= i", rec2, zbool(true))

	testSuccessful(t, "f = u", rec2, zbool(false))
	testSuccessful(t, "f != u", rec2, zbool(true))
	testSuccessful(t, "f < u", rec2, zbool(true))
	testSuccessful(t, "f <= u", rec2, zbool(true))
	testSuccessful(t, "f > u", rec2, zbool(false))
	testSuccessful(t, "f >= u", rec2, zbool(false))

	testSuccessful(t, "u = f", rec2, zbool(false))
	testSuccessful(t, "u != f", rec2, zbool(true))
	testSuccessful(t, "u < f", rec2, zbool(false))
	testSuccessful(t, "u <= f", rec2, zbool(false))
	testSuccessful(t, "u > f", rec2, zbool(true))
	testSuccessful(t, "u >= f", rec2, zbool(true))
}

func TestCompareNonNumbers(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[b:bool,s:string,bs:bstring,i:ip,p:port,net:net,t:time,d:duration]
0:[t;hello;world;10.1.1.1;443;10.1.0.0/16;1583794452;1000;]`)
	require.NoError(t, err)

	// bool
	testSuccessful(t, "b = true", record, zbool(true))
	testSuccessful(t, "b = false", record, zbool(false))
	testSuccessful(t, "b != true", record, zbool(false))
	testSuccessful(t, "b != false", record, zbool(true))

	// string
	testSuccessful(t, `s = "hello"`, record, zbool(true))
	testSuccessful(t, `s != "hello"`, record, zbool(false))
	testSuccessful(t, `s = "world"`, record, zbool(false))
	testSuccessful(t, `s != "world"`, record, zbool(true))
	testSuccessful(t, `bs = "world"`, record, zbool(true))
	testSuccessful(t, `bs != "world"`, record, zbool(false))
	testSuccessful(t, `bs = "hello"`, record, zbool(false))
	testSuccessful(t, `bs != "hello"`, record, zbool(true))
	testSuccessful(t, "s = bs", record, zbool(false))
	testSuccessful(t, "s != bs", record, zbool(true))

	// ip
	testSuccessful(t, "i = 10.1.1.1", record, zbool(true))
	testSuccessful(t, "i != 10.1.1.1", record, zbool(false))
	testSuccessful(t, "i = 1.1.1.10", record, zbool(false))
	testSuccessful(t, "i != 1.1.1.10", record, zbool(true))
	testSuccessful(t, "i = i", record, zbool(true))

	// port
	testSuccessful(t, "p = 443", record, zbool(true))
	testSuccessful(t, "p != 443", record, zbool(false))
	testSuccessful(t, "p = :443", record, zbool(true))
	testSuccessful(t, "p != :443", record, zbool(false))

	// net
	testSuccessful(t, "net = 10.1.0.0/16", record, zbool(true))
	testSuccessful(t, "net != 10.1.0.0/16", record, zbool(false))
	testSuccessful(t, "net = 10.1.0.0/24", record, zbool(false))
	testSuccessful(t, "net != 10.1.0.0/24", record, zbool(true))

	// Test comparisons between incompatible types
	allTypes := []struct {
		field string
		typ   string
	}{
		{"b", "bool"},
		{"s", "string"},
		{"bs", "bstring"},
		{"i", "ip"},
		{"p", "port"},
		{"net", "net"},
	}

	allOperators := []string{"=", "!=", "<", "<=", ">", ">="}

	for _, t1 := range allTypes {
		for _, t2 := range allTypes {
			if t1 == t2 || (t1.typ == "string" && t2.typ == "bstring") || (t1.typ == "bstring" && t2.typ == "string") {
				continue
			}
			for _, op := range allOperators {
				exp := fmt.Sprintf("%s = %s", t1.field, t2.field)
				desc := fmt.Sprintf("compare %s %s %s", t1.typ, op, t2.typ)
				testError(t, exp, record, expr.ErrIncompatibleTypes, desc)
			}
		}
	}

	// relative comparisons on strings
	record, err = parseOneRecord(`
#0:record[s:string,bs:bstring]
0:[abc;def;]`)
	require.NoError(t, err)

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

	testSuccessful(t, `bs < "security"`, record, zbool(true))
	testSuccessful(t, `bs < "aaa"`, record, zbool(false))
	testSuccessful(t, `bs < "def"`, record, zbool(false))

	testSuccessful(t, `bs > "security"`, record, zbool(false))
	testSuccessful(t, `bs > "aaa"`, record, zbool(true))
	testSuccessful(t, `bs > "def"`, record, zbool(false))

	testSuccessful(t, `bs <= "security"`, record, zbool(true))
	testSuccessful(t, `bs <= "aaa"`, record, zbool(false))
	testSuccessful(t, `bs <= "def"`, record, zbool(true))

	testSuccessful(t, `bs >= "security"`, record, zbool(false))
	testSuccessful(t, `bs >= "aaa"`, record, zbool(true))
	testSuccessful(t, `bs >= "def"`, record, zbool(true))
}

func TestArithmetic(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[x:int32,f:float64]
0:[10;2.5;]`)
	require.NoError(t, err)

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

	// Test arithmetic between integer types
	iresult := func(t1, t2 string, v uint64) zng.Value {
		if (t1[0] == 'u' || t1 == "byte") && (t2[0] == 'u' || t2 == "byte") {
			return zuint64(v)
		} else {
			return zint64(int64(v))
		}
	}

	var intTypes = []string{"byte", "int16", "uint16", "int32", "uint32", "int64", "uint64"}
	for _, t1 := range intTypes {
		for _, t2 := range intTypes {
			record, err = parseOneRecord(fmt.Sprintf(`
#0:record[a:%s,b:%s]
0:[4;2;]`, t1, t2))
			require.NoError(t, err)

			testSuccessful(t, "a + b", record, iresult(t1, t2, 6))
			testSuccessful(t, "b + a", record, iresult(t1, t2, 6))
			testSuccessful(t, "a - b", record, iresult(t1, t2, 2))
			testSuccessful(t, "a * b", record, iresult(t1, t2, 8))
			testSuccessful(t, "b * a", record, iresult(t1, t2, 8))
			testSuccessful(t, "a / b", record, iresult(t1, t2, 2))
			testSuccessful(t, "b / a", record, iresult(t1, t2, 0))
		}

		// Test arithmetic mixing float + int
		record, err = parseOneRecord(fmt.Sprintf(`
#0:record[x:%s,f:float64]
0:[10;2.5;]`, t1))
		require.NoError(t, err)

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
	testError(t, `"hello" - " world"`, record, expr.ErrIncompatibleTypes, "subtracting strings")
	testError(t, `"hello" * " world"`, record, expr.ErrIncompatibleTypes, "multiplying strings")
	testError(t, `"hello" / " world"`, record, expr.ErrIncompatibleTypes, "dividing strings")

	// Test that addition fails on an unsupported type
	testError(t, "10.1.1.1 + 1", record, expr.ErrIncompatibleTypes, "adding ip and integer")
	testError(t, "10.1.1.1 + 3.14159", record, expr.ErrIncompatibleTypes, "adding ip and float")
	testError(t, `10.1.1.1 + "foo"`, record, expr.ErrIncompatibleTypes, "adding ip and string")
}

func TestArrayIndex(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[x:array[int64],i:uint16]
0:[[1;2;3;]1;]`)
	require.NoError(t, err)

	testSuccessful(t, "x[0]", record, zint64(1))
	testSuccessful(t, "x[1]", record, zint64(2))
	testSuccessful(t, "x[2]", record, zint64(3))
	testSuccessful(t, "x[i]", record, zint64(2))
	testSuccessful(t, "i+1", record, zint64(2))
	testSuccessful(t, "x[i+1]", record, zint64(3))

	testError(t, "x[-1]", record, expr.ErrIndexOutOfBounds, "negative array index")
	testError(t, "x[3]", record, expr.ErrIndexOutOfBounds, "array index too large")
}

func TestFieldReference(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[rec:record[i:int32,s:string,f:float64]]
0:[[5;boo;6.1;]]`)
	require.NoError(t, err)

	testSuccessful(t, "rec.i", record, zint32(5))
	testSuccessful(t, "rec.s", record, zstring("boo"))
	testSuccessful(t, "rec.f", record, zfloat64(6.1))

	testError(t, "rec.no", record, expr.ErrNoSuchField, "referencing nonexistent field")
}

func TestConditional(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[x:int64]
0:[1;]`)
	require.NoError(t, err)

	testSuccessful(t, `x = 0 ? "zero" : "not zero"`, record, zstring("not zero"))
	testSuccessful(t, `x = 1 ? "one" : "not one"`, record, zstring("one"))
	testError(t, `x ? "x" : "not x"`, record, expr.ErrIncompatibleTypes, "conditional with non-boolean condition")

	// Ensure that the unevaluated clause doesn't generate errors
	// (field y doesn't exist but it shouldn't be evaluated)
	testSuccessful(t, "x = 0 ? y : x", record, zint64(1))
	testSuccessful(t, "x != 0 ? x : y", record, zint64(1))
}
