package groupby

import (
	"errors"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type valRow []reducer.Interface

func newValRow(zctx *resolver.Context, makers []reducer.Maker) valRow {
	cols := make([]reducer.Interface, 0, len(makers))
	for _, maker := range makers {
		cols = append(cols, maker(zctx))
	}
	return cols
}

func (v valRow) consume(rec *zng.Record) {
	for _, r := range v {
		r.Consume(rec)
	}
}

func (v valRow) consumePartial(rec *zng.Record, vals []expr.Evaluator) error {
	for k, r := range v {
		dec, ok := r.(reducer.Decomposable)
		if !ok {
			return errors.New("reducer row doesn't decompose")
		}
		v, err := vals[k].Eval(rec)
		if err != nil {
			return err
		}
		if err := dec.ConsumePart(v); err != nil {
			return err
		}
	}
	return nil
}
