package op

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type applier struct {
	octx   *Context
	parent zbuf.Puller
	expr   expr.Evaluator
}

func NewApplier(octx *Context, parent zbuf.Puller, expr expr.Evaluator) *applier {
	return &applier{
		octx:   octx,
		parent: parent,
		expr:   expr,
	}
}

func (a *applier) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull(done)
		if batch == nil || err != nil {
			return nil, err
		}
		vals := batch.Values()
		out := make([]zed.Value, 0, len(vals))
		for i := range vals {
			val := a.expr.Eval(batch, vals[i])
			if val.IsError() {
				if val.IsQuiet() || val.IsMissing() {
					continue
				}
			}
			out = append(out, val)
		}
		if len(out) > 0 {
			defer batch.Unref()
			return zbuf.NewBatch(batch, out), nil
		}
		batch.Unref()
	}
}
