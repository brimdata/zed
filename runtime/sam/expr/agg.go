package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr/agg"
)

type Aggregator struct {
	pattern agg.Pattern
	expr    Evaluator
	where   Evaluator
}

func NewAggregator(op string, expr Evaluator, where Evaluator) (*Aggregator, error) {
	pattern, err := agg.NewPattern(op, expr != nil)
	if err != nil {
		return nil, err
	}
	if expr == nil {
		// Count is the only that has no argument so we just return
		// true so it counts each value encountered.
		expr = &Literal{zed.True}
	}
	return &Aggregator{
		pattern: pattern,
		expr:    expr,
		where:   where,
	}, nil
}

func (a *Aggregator) NewFunction(zctx *zed.Context) agg.Function {
	return a.pattern(zctx)
}

func (a *Aggregator) Apply(ectx Context, f agg.Function, this zed.Value) {
	if a.where != nil {
		if val, ok := EvalBool(ectx, this, a.where); !ok || !val.Bool() {
			// XXX Issue #3401: do something with "where" errors.
			return
		}
	}
	v := a.expr.Eval(ectx, this)
	if !v.IsMissing() {
		f.Consume(v)
	}
}

// NewAggregatorExpr returns an Evaluator from agg. The returned Evaluator
// retains the same functionality of the aggregation only it returns it's
// current state every time a new value is consumed.
func NewAggregatorExpr(zctx *zed.Context, agg *Aggregator) Evaluator {
	return &aggregatorExpr{agg: agg, zctx: zctx}
}

type aggregatorExpr struct {
	agg  *Aggregator
	fn   agg.Function
	zctx *zed.Context
}

var _ Evaluator = (*aggregatorExpr)(nil)

func (s *aggregatorExpr) Eval(ectx Context, val zed.Value) zed.Value {
	if s.fn == nil {
		s.fn = s.agg.NewFunction(s.zctx)
	}
	s.agg.Apply(ectx, s.fn, val)
	return s.fn.Result(ectx.Arena())
}
