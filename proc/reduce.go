package proc

import (
	"github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type ReduceParams struct {
	reducers []compile.CompiledReducer
}

type Reduce struct {
	Base
	n       int
	columns compile.Row
}

func NewReduce(c *Context, parent Proc, params ReduceParams) Proc {

	return &Reduce{
		Base:    Base{Context: c, Parent: parent},
		columns: compile.Row{Defs: params.reducers},
	}
}

func (r *Reduce) output() (*zbuf.Array, error) {
	rec, err := r.columns.Result(r.Context.TypeContext)
	if err != nil {
		return nil, err
	}
	return zbuf.NewArray([]*zng.Record{rec}), nil
}

func (r *Reduce) Pull() (zbuf.Batch, error) {
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
		return r.output()
	}
	for k := 0; k < batch.Length(); k++ {
		r.consume(batch.Index(k))
	}
	batch.Unref()
	return zbuf.NewArray([]*zng.Record{}), nil
}

func (r *Reduce) consume(rec *zng.Record) {
	r.n++
	r.columns.Consume(rec)
}
