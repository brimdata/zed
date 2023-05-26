package semantic

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/lakeparse"
)

// Analyze performs a semantic analysis of the AST, translating it from AST
// to DAG form, resolving syntax ambiguities, and performing constant propagation.
// After semantic analysis, the DAG is ready for either optimization or compilation.
func Analyze(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish) (dag.Seq, error) {
	c := newAnalyzer(ctx, source, head)
	return c.check(seq)
}

type analyzer struct {
	ctx    context.Context
	source *data.Source
	scope  *Scope
	head   *lakeparse.Commitish
	zctx   *zed.Context
}

func newAnalyzer(ctx context.Context, source *data.Source, head *lakeparse.Commitish) *analyzer {
	return &analyzer{
		ctx:    ctx,
		source: source,
		scope:  NewScope(),
		head:   head,
		zctx:   zed.NewContext(),
	}
}

func (c *analyzer) check(seq ast.Seq) (dag.Seq, error) {
	return c.semSeq(seq)
}
