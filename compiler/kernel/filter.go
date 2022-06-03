package kernel

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type Filter struct {
	builder  *Builder
	pushdown dag.Expr
}

var _ zbuf.Filter = (*Filter)(nil)

func (f *Filter) AsEvaluator() (expr.Evaluator, error) {
	if f == nil {
		return nil, nil
	}
	return f.builder.compileExpr(f.pushdown)
}

func (f *Filter) AsBufferFilter() (*expr.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileBufferFilter(f.builder.pctx.Zctx, f.pushdown)
}

func (f *Filter) AsKeySpanFilter(key field.Path, o order.Which) (*expr.SpanFilter, error) {
	k := f.KeyFilter(key)
	if k == nil {
		return nil, nil
	}
	e := k.SpanFilter(o)
	eval, err := CompileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewSpanFilter(eval), nil
}

func (f *Filter) AsKeyCroppedByFilter(key field.Path, o order.Which) (*expr.SpanFilter, error) {
	k := f.KeyFilter(key)
	if k == nil {
		return nil, nil
	}
	e := k.CroppedByFilter(o)
	eval, err := CompileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewSpanFilter(eval), nil
}

func (f *Filter) KeyFilter(key field.Path) *dag.KeyFilter {
	if f == nil {
		return nil
	}
	return dag.NewKeyFilter(key, f.pushdown)
}

func (f *Filter) Pushdown() dag.Expr {
	if f == nil {
		return nil
	}
	return f.pushdown
}
