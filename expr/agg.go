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

func (a *Aggregator) Apply(f agg.Function, rec *zed.Value) error {
	if a.filter(rec) {
		return nil
	}
	zv, err := a.expr.Eval(rec)
	if err != nil {
		if err == zed.ErrMissing {
			err = nil
		}
		return err
	}
	return f.Consume(zv)
}

func (a *Aggregator) filter(rec *zed.Value) bool {
	if a.where == nil {
		return false
	}
	return !a.where(rec)
}
