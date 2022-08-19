package dag_test

import (
	"context"
	"testing"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/zfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyFilter(t *testing.T) {
	test := func(query, expected string, t *testing.T) {
		t.Run(query, func(t *testing.T) {

			p := compiler.MustParse(query)
			op, err := semantic.Analyze(context.Background(), p.(*ast.Sequential), nil, nil)
			require.NoError(t, err)
			kf := dag.NewKeyFilter(field.New("pk"), op.Ops[0].(*dag.Filter).Expr)
			if kf == nil {
				assert.Equal(t, expected, "", "expected key filter to be optimizable but it was not")
				return
			}
			assert.Equal(t, expected, zfmt.DAGExpr(kf.Expr))
		})
	}
	test("pk<1", "pk<1", t)
	test("pk<1 and foo==\"bar\"", "pk<1", t)
	test("pk<1 or pk>3", "pk<1 or pk>3", t)
	test("pk<1 or foo==\"bar\"", "", t)
	test("(pk>1 and pk<3) or (pk>4 and pk<6)", "pk>1 and pk<3 or pk>4 and pk<6", t)
	test("pk==1 and pk ==1 and (pk==3 or foo==\"bar\")", "", t)
}
