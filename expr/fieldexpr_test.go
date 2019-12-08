package expr_test

import (
	"testing"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldExprString(t *testing.T) {
	testcase(t, "id")
	testcase(t, "id.resp_h")
	testcase(t, "id.resp_h[0]")
	testcase(t, "a.b.c.d.e.f")
}

func testcase(t *testing.T, fieldexpr string) {
	p, err := zql.Parse("", []byte(fieldexpr+"=64"))
	require.NoError(t, err)

	field := p.(*ast.FilterProc).Filter.(*ast.CompareField).Field
	str, err := expr.FieldExprString(field)
	require.NoError(t, err)
	assert.Equal(t, fieldexpr, str)
}
