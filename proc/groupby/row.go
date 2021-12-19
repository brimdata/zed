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

func (v valRow) apply(aggs []*expr.Aggregator, this *zed.Value, scope *expr.Scope) {
	for k, a := range aggs {
		a.Apply(v[k], this, scope)
	}
}

func (v valRow) consumeAsPartial(rec *zed.Value, exprs []expr.Evaluator, scope *expr.Scope) {
	for k, r := range v {
		val := exprs[k].Eval(rec, scope)
		//XXX should do soemthing with errors... they could come from
		// a worker over the network?
		if !val.IsError() {
			r.ConsumeAsPartial(val)
		}
	}
}
