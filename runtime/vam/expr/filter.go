package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

// yield what(this) where where(this)
// when expr is This{}, we get "where what(this)"
type filter struct {
	what  Evaluator
	where Evaluator
}

// A filter evaluates an expression to produce a result, applies
// predicate to that result, then
func NewFilter(what, where Evaluator) Evaluator {
	// commpiler should wrap filter in apply so we don't
	// need to run the apply inside here (fix sam side)
	return &filter{what, where}
}

func (f *filter) Eval(this vector.Any) (vector.Any, *vector.Error) {
	val, verr := f.what.Eval(this)
	//XXX
	if val == nil {
		return nil, verr
	}
	where := f.where.Eval(val)
	// XXX check that where is a bool and select the slots from
	// val indicated by where... and skip and propagate errors
	return zed.False
}

// XXX selection
type filterApplier struct {
	zctx *zed.Context
	expr Evaluator
}

func NewFilterApplier(zctx *zed.Context, e Evaluator) Evaluator {
	return &filterApplier{zctx, e}
}

func (f *filterApplier) Eval(this vector.Any) (vector.Any, *vector.Error) {
	val, ok := EvalBool(f.zctx, this, f.expr)
	if ok {
		if val.Bool() {
			return this
		}
		return f.zctx.Missing()
	}
	return val
}

type Apply struct {
	e Evaluator
}

func (a *Apply) Eval(val vector.Any) (vector.Any, *vector.Error) {
	//XXX if val is invariant we can do generic unravel
	return a.Eval(val)
}

// yield expr(this) where pred(this)
