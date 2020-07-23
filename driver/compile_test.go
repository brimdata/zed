package driver

import (
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeColumns(t *testing.T) {
	tests := []struct {
		zql  string
		cols []string
	}{
		{
			"cut x, y, z",
			[]string{"x", "y", "z"},
		},
		{
			"cut -c foo | cut x, y, z",
			[]string{"x", "y", "z"},
		},
		{
			"put x=y | filter x > 1",
			nil,
		},
		{
			"put x=y | filter x > 1 | cut x",
			[]string{"x", "y"},
		},
		{
			"head 1 | tail 1 | cut x, y, z",
			[]string{"x", "y", "z"},
		},
		{
			"count()",
			[]string{},
		},
		{
			"count(y) by x",
			[]string{"x", "y"},
		},
		{
			"every 1s count(y) by x",
			[]string{"ts", "x", "y"},
		},
		{
			"every 1s count(y) by foo=String.replace(x, y, z)",
			[]string{"ts", "x", "y", "z"},
		},
		{
			"filter x=1 | every 1s count(y) by foo=z | (head 1; tail 1)",
			[]string{"ts", "x", "y", "z"},
		},
		{
			"put x=1 | every 1s count(y) by foo=z | (head 1; tail 1)",
			[]string{"ts", "y", "z"},
		},
		{
			"rename foo=x | every 1s count(y) by foo=z | (head 1; tail 1)",
			[]string{"ts", "x", "y", "z"},
		},
		{
			"*>1 | every 1s count(y) by foo=String.replace(x, y, z) | (head 1; tail 1)",
			nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.zql, func(t *testing.T) {
			query, err := zql.ParseProc(tc.zql)
			require.NoError(t, err)
			// apply ReplaceGroupByProcDurationWithKey here because computeColumns
			// will be applied after it when it is plugged into compilation.
			ReplaceGroupByProcDurationWithKey(query)
			cols := computeColumns(query)
			var expected map[string]struct{}
			if tc.cols != nil {
				expected = make(map[string]struct{})
				for _, s := range tc.cols {
					expected[s] = struct{}{}
				}
			}
			assert.Equal(t, expected, cols)
		})
	}
}

func TestExpressionFields(t *testing.T) {
	tests := []struct {
		expr     string
		expected []string
	}{
		{
			"a=1",
			[]string{"a"},
		},
		{
			"a:time",
			[]string{"a"},
		},
		{
			"!a=1",
			[]string{"a"},
		},
		{
			"a=1 or b=c",
			[]string{"a", "b", "c"},
		},
		{
			"a ? b : c + d",
			[]string{"a", "b", "c", "d"},
		},
		{
			"Time.trunc(ts, 10)",
			[]string{"ts"},
		},
		{
			"Math.max(a, b, c, d)",
			[]string{"a", "b", "c", "d"},
		},
		{
			"String.replace(a, b, c)",
			[]string{"a", "b", "c"},
		},
		{
			"String.replace(a, 'b', c)",
			[]string{"a", "c"},
		},
		{
			"x = 0 ? y : z",
			[]string{"x", "y", "z"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			parsed, err := zql.Parse("", []byte(tc.expr), zql.Entrypoint("Expression"))
			require.NoError(t, err)
			f := expressionFields(parsed.(ast.Expression))
			assert.Equal(t, tc.expected, f)
		})
	}
}

func TestBooleanExpressionFields(t *testing.T) {
	tests := []struct {
		expr     string
		expected []string
	}{
		{
			"* = 1",
			nil,
		},
		{
			"*",
			[]string{},
		},
		{
			"* OR *=1",
			nil,
		},
		{
			"* OR x=1",
			[]string{"x"},
		},
		{
			"!(* OR x=1)",
			[]string{"x"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			parsed, err := zql.Parse("", []byte(tc.expr), zql.Entrypoint("searchExpr"))
			require.NoError(t, err)
			f := booleanExpressionFields(parsed.(ast.BooleanExpr))
			assert.Equal(t, tc.expected, f)
		})
	}
}
