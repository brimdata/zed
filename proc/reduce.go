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
	params ReduceParams
}

func NewReduce(c *Context, parent Proc, params ReduceParams) Proc {
	return &Reduce{
		Base:   Base{Context: c, Parent: parent},
		params: params,
	}
}

func (r *Reduce) Pull() (zbuf.Batch, error) {
	var columns compile.Row
	var consumed bool
	for {
		batch, err := r.Get()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			break
		}
		if !consumed {
			consumed = true
			columns = compile.NewRow(r.params.reducers)
		}
		for k := 0; k < batch.Length(); k++ {
			columns.Consume(batch.Index(k))
		}
		batch.Unref()
	}
	if !consumed {
		return nil, nil
	}
	rec, err := columns.Result(r.Context.TypeContext)
	if err != nil {
		return nil, err
	}
	return zbuf.NewArray([]*zng.Record{rec}), nil
}
