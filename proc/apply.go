package proc

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
)

type applier struct {
	pctx   *Context
	parent Interface
	expr   expr.Applier
	warned map[string]struct{}
}

func NewApplier(pctx *Context, parent Interface, apply expr.Applier) *applier {
	return &applier{
		pctx:   pctx,
		parent: parent,
		expr:   apply,
		warned: make(map[string]struct{}),
	}
}

func (a *applier) Pull() (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull()
		if batch == nil || err != nil {
			return nil, err
		}
		ectx := batch.Context()
		vals := batch.Values()
		out := make([]zed.Value, 0, len(vals))
		for i := range vals {
			val := a.expr.Eval(ectx, &vals[i])
			if val.IsError() {
				if val.IsQuiet() || val.IsMissing() {
					continue
				}
			}
			// Copy is necessary because Apply can return
			// its argument.
			out = append(out, *val.Copy())
		}
		batch.Unref()
		if len(out) > 0 {
			//XXX bug - need to propagate scope
			return zbuf.NewArray(out), nil
		}
	}
}

func (a *applier) Done() {
	a.parent.Done()
}
