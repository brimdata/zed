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
	tmp    []vector.Any
}

type valerr struct {
	val vector.Any
	err vector.Any
}

var _ vector.Puller = (*Yield)(nil)

func NewYield(zctx *zed.Context, parent vector.Puller, exprs []expr.Evaluator) *Yield {
	return &Yield{
		parent: parent,
		exprs:  exprs,
	}
}

func (y *Yield) Pull(done bool) (vector.Any, error) {
	for {
		val, err := o.parent.Pull(done)
		if val == nil {
			return nil, err
		}
		//XXX this currently only works if order doesn't matter
		// can put together with a union or if they are the same
		// types then put together with an interleave
		// e.g., yield x, x+1 needs to interleave x[0],x[0]+1,x[1],x[1]+1
		for _, e := range y.exprs {
			val, err := e.Eval(val, vals[i])
			//XXX need to quiet by row... which is just a filter step
			// with a quiet boolean
			//if val.IsQuiet() {
			//	continue
			//}
			out = append(out, val.Copy())
		}
	}
}

func apply(e expr.Evaluator, val, err vector.Any) (vector.Any, vector.Any) {
	val, newErr := e.Eval(val)
	if newErr != nil {
		err = mixErr(err, newErr)
	}
	return val, err
}

func mixErr(e0, e1 vector.Any) vector.Any {
	if e0 == nil {
		return e1
	}
	if e1 == nil {
		return e0
	}
	//XXX
	panic("vector runtime: no support yet for stacked errors")
}
