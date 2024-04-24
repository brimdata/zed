package yield

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
	parent zbuf.Puller
	exprs  []expr.Evaluator
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
		arena := zed.NewArena()
		ectx := expr.NewContextWithVars(arena, batch.Vars())
		vals := batch.Values()
		out := make([]zed.Value, 0, len(o.exprs)*len(vals))
		for i := range vals {
			for _, e := range o.exprs {
				val := e.Eval(ectx, vals[i])
				if val.IsQuiet() {
					continue
				}
				out = append(out, val)
			}
		}
		if len(out) > 0 {
			defer arena.Unref()
			defer batch.Unref()
			return zbuf.NewBatch(arena, out, batch, batch.Vars()), nil
		}
		arena.Unref()
		batch.Unref()
	}
}
