package zngio

import (
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredPattern(t *testing.T) {
	literal := ast.Literal{Type: "string", Value: "value"}
	longerLiteral := ast.Literal{Type: "string", Value: "longervalue"}
	cases := []struct {
		input    ast.BooleanExpr
		expected string
	}{
		{&ast.CompareAny{Comparator: "<", Value: literal}, ""},
		{&ast.CompareAny{Comparator: "=", Value: literal}, "\fvalue"},
		{&ast.CompareAny{Comparator: "in", Value: literal}, "\fvalue"},
		{&ast.CompareField{Comparator: "<", Value: literal}, ""},
		{&ast.CompareField{Comparator: "=", Value: literal}, "\fvalue"},
		{&ast.CompareField{Comparator: "in", Value: literal}, "\fvalue"},
		{&ast.LogicalAnd{
			Left:  &ast.CompareAny{Comparator: "=", Value: literal},
			Right: &ast.CompareAny{Comparator: "=", Value: longerLiteral},
		}, "\x18longervalue"},
		{&ast.LogicalAnd{
			Left:  &ast.CompareAny{Comparator: "=", Value: longerLiteral},
			Right: &ast.CompareAny{Comparator: "=", Value: literal},
		}, "\x18longervalue"},
		{&ast.LogicalNot{
			Expr: &ast.CompareAny{Comparator: "=", Value: literal},
		}, ""},
		{&ast.LogicalOr{
			Left:  &ast.CompareAny{Comparator: "=", Value: literal},
			Right: &ast.CompareAny{Comparator: "=", Value: longerLiteral},
		}, ""},
		{&ast.MatchAll{}, ""},
		{&ast.Search{Text: literal.Value, Value: literal}, ""},
	}
	for i, c := range cases {
		p, err := requiredPattern(c.input)
		require.NoError(t, err)
		assert.Exactly(t, c.expected, p, "case %d", i)
	}
}
