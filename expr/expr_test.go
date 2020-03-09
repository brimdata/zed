package expr_test

import (
	"errors"
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

		assert.Equal(t, result.Type, expect.Type, "result type is correct")
		assert.Equal(t, result.Bytes, expect.Bytes, "result value is correct")
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

func zfloat64(f float64) zng.Value {
	return zng.Value{zng.TypeFloat64, zng.EncodeFloat64(f)}
}

func TestExpressions(t *testing.T) {
	TestPrimitives(t)
	TestLogical(t)
	TestArithmetic(t)
	// XXX test comparisons (equality + relative)
}

func TestPrimitives(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[x:int32,f:float64,i:ip]
0:[10;2.5;10.1.1.1;]`)
	require.NoError(t, err)

	// Test simple literals
	testSuccessful(t, "50", record, zint64(50))
	testSuccessful(t, "3.14", record, zfloat64(3.14))

	// Test good field references
	testSuccessful(t, "x", record, zint32(10))
	testSuccessful(t, "f", record, zfloat64(2.5))

	// Test bad field references
	_, err = evaluate("doesnexist", record)
	assert.Error(t, err, "referencing non-existent field gave an error")
	assert.Error(t, err, expr.ErrNoSuchField, "referencing non-existent field gave the correct error")
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

	// XXX associativity?
}

func TestArithmetic(t *testing.T) {
	// XXX should test all primitive int types
	record, err := parseOneRecord(`
#0:record[x:int32,f:float64,i:ip]
0:[10;2.5;10.1.1.1;]`)
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

	// Test arithmetic mixing float + int
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

	// Test that addition fails on an unsupported type
	_, err = evaluate("i + 1", record)
	assert.Error(t, err, "adding incompatible types gave an error")
	assert.Error(t, err, expr.ErrIncompatibleTypes, "adding incompatible types gave the correct error")

	_, err = evaluate("x + i", record)
	assert.Error(t, err, "adding incompatible types gave an error")
	assert.Error(t, err, expr.ErrIncompatibleTypes, "adding incompatible types gave the correct error")

	_, err = evaluate("f + i", record)
	assert.Error(t, err, "adding incompatible types gave an error")
	assert.Error(t, err, expr.ErrIncompatibleTypes, "adding incompatible types gave the correct error")
}
