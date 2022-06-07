package index

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zcode"
)

type filterEval func(*zed.Value) bool

func compileFilter(e dag.Expr, key field.Path, o order.Which) (*expr.SpanFilter, expr.Evaluator, error) {
	kf := dag.NewKeyFilter(key, e)
	if kf == nil {
		return nil, nil, errors.New("nothing to filter from expr")
	}
	spanFilter, err := compileSpanFilter(kf, o)
	if err != nil {
		return nil, nil, err
	}
	valFilter, err := kernel.CompileExpr(kf.Expr)
	if err != nil {
		return nil, nil, err
	}
	return spanFilter, valFilter, nil
}

type alloc struct {
	val zed.Value
}

func (a *alloc) NewValue(typ zed.Type, b zcode.Bytes) *zed.Value {
	a.val.Type = typ
	a.val.Bytes = append(a.val.Bytes[:0], b...)
	return &a.val
}

func (a *alloc) CopyValue(val *zed.Value) *zed.Value {
	a.val.Type = val.Type
	a.val.Bytes = append(a.val.Bytes[:0], val.Bytes...)
	return &a.val
}

func (a *alloc) Vars() []zed.Value {
	return nil
}

func compileSpanFilter(kf *dag.KeyFilter, o order.Which) (*expr.SpanFilter, error) {
	e := kf.SpanFilter(o)
	eval, err := kernel.CompileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewSpanFilter(eval), nil
}
