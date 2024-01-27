package op

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/vector"
)

type Yield struct {
	zctx   *zed.Context
	parent vector.Puller
	exprs  []expr.Evaluator
}

var _ vector.Puller = (*Yield)(nil)

func NewYield(zctx *zed.Context, parent vector.Puller, exprs []expr.Evaluator) *Yield {
	return &Yield{
		zctx:   zctx,
		parent: parent,
		exprs:  exprs,
	}
}

func (y *Yield) Pull(done bool) (vector.Any, error) {
	for {
		val, err := y.parent.Pull(done)
		if val == nil {
			return nil, err
		}
		//XXX this currently only works if order doesn't matter
		// can put together with a union or if they are the same
		// types then put together with an interleave
		// e.g., yield x, x+1 needs to interleave x[0],x[0]+1,x[1],x[1]+1
		for _, e := range y.exprs {
			v := e.Eval(val)
			//XXX need to quiet by row... which is just a filter step
			// with a quiet boolean
			//if val.IsQuiet() {
			//	continue
			//}
			//XXX need to interleve results
			return v, nil
		}
	}
}
