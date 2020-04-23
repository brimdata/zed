package proc

import (
	"time"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type ReducerParams struct {
	interval ast.Duration
	reducers []compile.CompiledReducer
}

type Reducer struct {
	Base
	n        int
	interval time.Duration
	columns  compile.Row
}

func NewReducer(c *Context, parent Proc, params ReducerParams) Proc {
	interval := time.Duration(params.interval.Seconds) * time.Second
	return &Reducer{
		Base:     Base{Context: c, Parent: parent},
		interval: interval,
		columns:  compile.Row{Defs: params.reducers},
	}
}

func (r *Reducer) output() *zbuf.Array {
	rec := r.columns.Result(r.Context.TypeContext)
	return zbuf.NewArray([]*zng.Record{rec}, nano.NewSpanTs(r.MinTs, r.MaxTs))
}

func (r *Reducer) Pull() (zbuf.Batch, error) {
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
	return zbuf.NewArray([]*zng.Record{}, batch.Span()), nil
}

func (r *Reducer) consume(rec *zng.Record) {
	r.n++
	r.columns.Consume(rec)
}
