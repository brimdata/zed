package proc

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
)

type applier struct {
	pctx   *Context
	parent zbuf.Puller
	expr   expr.Applier
	warned map[string]struct{}
}

func NewApplier(pctx *Context, parent zbuf.Puller, apply expr.Applier) *applier {
	return &applier{
		pctx:   pctx,
		parent: parent,
		expr:   apply,
		warned: make(map[string]struct{}),
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
			val := a.expr.Eval(batch, &vals[i])
			if val.IsError() {
				if val.IsQuiet() || val.IsMissing() {
					continue
				}
			}
			// Copy is necessary because Apply can return
			// its argument.
			out = append(out, *val.Copy())
		}
		if len(out) > 0 {
			defer batch.Unref()
			return zbuf.NewBatch(batch, out), nil
		}
		batch.Unref()
	}
}
