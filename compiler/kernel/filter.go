package kernel

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type Filter struct {
	Pushdown dag.Expr
	builder  *Builder
}

var _ zbuf.Filter = (*Filter)(nil)

func (f *Filter) AsEvaluator() (expr.Evaluator, error) {
	if f == nil {
		return nil, nil
	}
	return f.builder.compileExpr(f.Pushdown)
}

func (f *Filter) AsBufferFilter() (*expr.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileBufferFilter(f.builder.pctx.Zctx, f.Pushdown)
}
