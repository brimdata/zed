package proc

import (
	"time"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer/compile"
)

type ReducerProc struct {
	Base
	n        int
	interval time.Duration
	columns  compile.Row
}

func NewReducerProc(c *Context, parent Proc, params *ast.ReducerProc) Proc {
	interval := time.Duration(params.UpdateInterval.Seconds) * time.Second
	return &ReducerProc{
		Base:     Base{Context: c, Parent: parent},
		interval: interval,
		columns:  compile.Row{Defs: params.Reducers},
	}
}

func (r *ReducerProc) output() *zson.Array {
	rec := r.columns.Result(r.Context.Resolver)
	return zson.NewArray([]*zson.Record{rec}, nano.NewSpanTs(r.MinTs, r.MaxTs))
}

func (r *ReducerProc) Pull() (zson.Batch, error) {
	start := time.Now()
	for {
		batch, err := r.Get()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			//XXX why does this crash if we take out this test?
			if r.n == 0 {
				return nil, nil
			}
			r.n = 0
			return r.output(), nil
		}
		for k := 0; k < batch.Length(); k++ {
			r.consume(batch.Index(k))
		}
		batch.Unref()
		if r.interval > 0 && time.Since(start) >= r.interval {
			return r.output(), nil
		}
	}
}

func (r *ReducerProc) consume(rec *zson.Record) {
	r.n++
	r.columns.Consume(rec)
}
