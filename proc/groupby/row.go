package groupby

import (
	"github.com/brimdata/zq/expr"
	"github.com/brimdata/zq/expr/agg"
	"github.com/brimdata/zq/zng"
)

type valRow []agg.Function

func newValRow(aggs []*expr.Aggregator) valRow {
	cols := make([]agg.Function, 0, len(aggs))
	for _, a := range aggs {
		cols = append(cols, a.NewFunction())
	}
	return cols
}

func (v valRow) apply(aggs []*expr.Aggregator, rec *zng.Record) error {
	for k, a := range aggs {
		if err := a.Apply(v[k], rec); err != nil {
			return err
		}
	}
	return nil
}

func (v valRow) consumeAsPartial(rec *zng.Record, vals []expr.Evaluator) error {
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
