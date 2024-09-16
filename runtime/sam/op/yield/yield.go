package yield

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
	parent   zbuf.Puller
	exprs    []expr.Evaluator
	resetter expr.Resetter
}

func New(parent zbuf.Puller, exprs []expr.Evaluator, resetter expr.Resetter) *Op {
	return &Op{
		parent:   parent,
		exprs:    exprs,
		resetter: resetter,
	}
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := o.parent.Pull(done)
		if batch == nil || err != nil {
			o.resetter.Reset()
			return nil, err
		}
		vals := batch.Values()
		out := make([]zed.Value, 0, len(o.exprs)*len(vals))
		for i := range vals {
			for _, e := range o.exprs {
				val := e.Eval(batch, vals[i])
				if val.IsQuiet() {
					continue
				}
				out = append(out, val.Copy())
			}
		}
		if len(out) > 0 {
			defer batch.Unref()
			return zbuf.NewBatch(batch, out), nil
		}
		batch.Unref()
	}
}
