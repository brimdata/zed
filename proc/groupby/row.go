package groupby

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/agg"
)

type valRow []agg.Function

func newValRow(aggs []*expr.Aggregator) valRow {
	cols := make([]agg.Function, 0, len(aggs))
	for _, a := range aggs {
		cols = append(cols, a.NewFunction())
	}
	return cols
}

func (v valRow) apply(aggs []*expr.Aggregator, rec *zed.Value) error {
	for k, a := range aggs {
		if err := a.Apply(v[k], rec); err != nil {
			return err
		}
	}
	return nil
}

func (v valRow) consumeAsPartial(rec *zed.Value, vals []expr.Evaluator) error {
	for k, r := range v {
		v, err := vals[k].Eval(rec)
		if err != nil {
			return err
		}
		if err := r.ConsumeAsPartial(v); err != nil {
			return err
		}
	}
	return nil
}
