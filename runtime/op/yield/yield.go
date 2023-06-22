package yield

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
	parent zbuf.Puller
	exprs  []expr.Evaluator
	ectx   expr.ResetContext
}

func New(parent zbuf.Puller, exprs []expr.Evaluator) *Op {
	return &Op{
		parent: parent,
		exprs:  exprs,
	}
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := o.parent.Pull(done)
		if batch == nil || err != nil {
			return nil, err
		}
		o.ectx.SetVars(batch.Vars())
		vals := batch.Values()
		out := make([]zed.Value, 0, len(o.exprs)*len(vals))
		for i := range vals {
			for _, e := range o.exprs {
				val := e.Eval(o.ectx.Reset(), &vals[i])
				if val.IsQuiet() {
					continue
				}
				out = append(out, *val.Copy())
			}
		}
		if len(out) > 0 {
			defer batch.Unref()
			return zbuf.NewBatch(batch, out), nil
		}
		batch.Unref()
	}
}
