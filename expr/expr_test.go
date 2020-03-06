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

func zint32(v int32) zng.Value {
	return zng.Value{zng.TypeInt32, zng.EncodeInt(int64(v))}
}

func zint64(v int64) zng.Value {
	return zng.Value{zng.TypeInt64, zng.EncodeInt(v)}
}

func TestExpressions(t *testing.T) {
	record, err := parseOneRecord(`
#0:record[x:int32,i:ip]
0:[10;10.1.1.1;]`)
	require.NoError(t, err)

	// Test simple literals
	testSuccessful(t, "50", record, zint64(50))
	testSuccessful(t, "x", record, zint32(10))

	// Test addition
	testSuccessful(t, "100 + 23", record, zint64(123))
	testSuccessful(t, "x + 5", record, zint64(15))

	// Test that addition fails on an unsupported type
	_, err = evaluate("i + 1", record)
	assert.Error(t, err, "adding incompatible types gave an error")
	assert.Error(t, err, expr.ErrIncompatibleTypes, "adding incompatible types gave the correct error")

	_, err = evaluate("x + i", record)
	assert.Error(t, err, "adding incompatible types gave an error")
	assert.Error(t, err, expr.ErrIncompatibleTypes, "adding incompatible types gave the correct error")
}
