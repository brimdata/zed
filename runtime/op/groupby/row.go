package groupby

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/agg"
	"github.com/brimdata/zed/zson"
)

type valRow []agg.Function

func newValRow(aggs []*expr.Aggregator) valRow {
	cols := make([]agg.Function, 0, len(aggs))
	for _, a := range aggs {
		cols = append(cols, a.NewFunction())
	}
	return cols
}

func (v valRow) apply(zctx *zed.Context, ectx expr.Context, aggs []*expr.Aggregator, this *zed.Value) {
	for k, a := range aggs {
		a.Apply(zctx, ectx, v[k], this)
	}
}

func (v valRow) consumeAsPartial(rec *zed.Value, exprs []expr.Evaluator, ectx expr.Context) {
	for k, r := range v {
		val := exprs[k].Eval(ectx, rec)
		if val.IsError() {
			panic(fmt.Errorf("consumeAsPartial: read a Zed error: %s", zson.MustFormatValue(val)))
		}
		//XXX should do soemthing with errors... they could come from
		// a worker over the network?
		if !val.IsError() {
			r.ConsumeAsPartial(val)
		}
	}
}
