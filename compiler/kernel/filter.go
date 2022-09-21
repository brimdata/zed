package kernel

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type Filter struct {
	pushdown dag.Expr
	builder  *Builder
}

var _ zbuf.Filter = (*Filter)(nil)

func (f *Filter) Pushdown() dag.Expr {
	if f == nil {
		return nil
	}
	return f.pushdown
}

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
	k := f.keyFilter(key)
	if k == nil {
		return nil, nil
	}
	e := k.SpanExpr(o)
	eval, err := compileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewSpanFilter(eval), nil
}

func (f *Filter) AsKeyCroppedByFilter(key field.Path, o order.Which) (*expr.SpanFilter, error) {
	k := f.keyFilter(key)
	if k == nil {
		return nil, nil
	}
	e := k.CroppedByExpr(o)
	eval, err := compileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewSpanFilter(eval), nil
}

func (f *Filter) keyFilter(key field.Path) *dag.KeyFilter {
	if f == nil {
		return nil
	}
	return dag.NewKeyFilter(key, f.pushdown)
}

type DeleteFilter struct {
	*Filter
}

func (f *DeleteFilter) AsEvaluator() (expr.Evaluator, error) {
	if f == nil {
		return nil, nil
	}
	// For a DeleteFilter Evaluator the pushdown gets wrapped in an unary
	// expression so we get all records that don't match.
	return f.builder.compileExpr(&dag.UnaryExpr{
		Kind:    "UnaryExpr",
		Op:      "!",
		Operand: f.pushdown,
	})
}

func (f *DeleteFilter) AsBufferFilter() (*expr.BufferFilter, error) {
	return nil, nil
}
