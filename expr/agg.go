package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/agg"
)

var (
	ErrBadValue      = errors.New("bad value")
	ErrFieldRequired = errors.New("field parameter required")
)

type Aggregator struct {
	pattern agg.Pattern
	expr    Evaluator
	where   Filter
}

func NewAggregator(op string, expr Evaluator, where Filter) (*Aggregator, error) {
	pattern, err := agg.NewPattern(op)
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

func (a *Aggregator) NewFunction() agg.Function {
	return a.pattern()
}

func (a *Aggregator) Apply(ectx Context, f agg.Function, val *zed.Value) {
	if a.filter(ectx, val) {
		return
	}
	v := a.expr.Eval(ectx, val)
	if !v.IsMissing() {
		f.Consume(v)
	}
}

func (a *Aggregator) filter(ectx Context, this *zed.Value) bool {
	if a.where == nil {
		return false
	}
	return !a.where(ectx, this)
}

// NewAggregatorExpr returns an Evaluator from agg. The returned Evaluator
// retains the same functionality of the aggregation only it returns it's
// current state every time a new value is consumed.
func NewAggregatorExpr(agg *Aggregator) Evaluator {
	return &aggregatorExpr{agg: agg}
}

type aggregatorExpr struct {
	agg  *Aggregator
	fn   agg.Function
	zctx *zed.Context
}

var _ Evaluator = (*aggregatorExpr)(nil)

func (s *aggregatorExpr) Eval(ectx Context, val *zed.Value) *zed.Value {
	if s.fn == nil {
		s.fn = s.agg.NewFunction()
		s.zctx = zed.NewContext()
	}
	s.agg.Apply(ectx, s.fn, val)
	return s.fn.Result(s.zctx)
}
