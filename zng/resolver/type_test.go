package resolver_test

import (
	"fmt"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zx"
	"github.com/stretchr/testify/require"
)

func val(t, v string) ast.Literal {
	return ast.Literal{t, v}
}

func runArray(f zx.Predicate, vals []ast.Literal, expected []bool) error {
	ctx := resolver.NewContext()
	for k, c := range vals {
		typ, err := ctx.LookupByName(c.Type)
		if err != nil {
			return err
		}
		zv, err := typ.Parse([]byte(c.Value))
		if err != nil {
			return err
		}
		value := zng.Value{typ, zv}
		if f(value) != expected[k] {
			return fmt.Errorf("value '%s' of type %s at slot %d failed test", c.Value, typ, k)
		}
	}
	return nil
}

func run(vals []ast.Literal, op string, v ast.Literal, results []bool) error {
	pred, err := zx.Comparison(op, v)
	if err != nil {
		return err
	}
	return runArray(pred, vals, results)
}

func TestZeek(t *testing.T) {
	t.Parallel()

	vals := []ast.Literal{
		{"int32", "100"},
		{"int32", "101"},
		{"float64", "100"},
		{"float64", "100.0"},
		{"float64", "100.5"},
		{"ip", "128.32.1.1"},
		{"string", "hello"},
		{"port", "80"},
		{"port", "8080"},
	}
	err := run(vals, "lt", val("int32", "101"), []bool{true, false, true, true, true, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "lte", val("int32", "101"), []bool{true, true, true, true, true, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "lte", val("float64", "100.2"), []bool{true, false, true, true, false, false, false, true, false})
	require.NoError(t, err)
	err = run(vals, "gt", val("port", "100"), []bool{false, false, false, false, false, false, false, false, true})
	require.NoError(t, err)
	err = run(vals, "eql", val("ip", "128.32.1.1"), []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", val("ip", "128.32.2.2"), []bool{false, false, false, false, false, false, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", val("net", "128.32.0.0/16"), []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", val("net", "128.32.0.0/16"), []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", val("net", "128.32.1.0/24"), []bool{false, false, false, false, false, true, false, false, false})
	require.NoError(t, err)
	err = run(vals, "eql", val("net", "128.32.2.0/24"), []bool{false, false, false, false, false, false, false, false, false})
	require.NoError(t, err)
}
