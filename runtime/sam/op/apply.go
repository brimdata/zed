package op

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zbuf"
)

type applier struct {
	rctx     *runtime.Context
	parent   zbuf.Puller
	expr     expr.Evaluator
	resetter expr.Resetter
}

func NewApplier(rctx *runtime.Context, parent zbuf.Puller, expr expr.Evaluator, resetter expr.Resetter) *applier {
	return &applier{
		rctx:     rctx,
		parent:   parent,
		expr:     expr,
		resetter: resetter,
	}
}

func (a *applier) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull(done)
		if batch == nil || err != nil {
			a.resetter.Reset()
			return nil, err
		}
		arena := zed.NewArena()
		ectx := expr.NewContextWithVars(arena, batch.Vars())
		vals := batch.Values()
		out := make([]zed.Value, 0, len(vals))
		for i := range vals {
			val := a.expr.Eval(ectx, vals[i])
			if val.IsError() {
				if val.IsQuiet() || val.IsMissing() {
					continue
				}
			}
			out = append(out, val)
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
