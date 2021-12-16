package proc

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
)

type Applier interface {
	expr.Evaluator
	fmt.Stringer
	Warning() string
}

type applier struct {
	pctx   *Context
	parent Interface
	expr   Applier
	warned map[string]bool
}

func NewApplier(pctx *Context, parent Interface, apply Applier) *applier {
	return &applier{
		pctx:   pctx,
		parent: parent,
		expr:   apply,
		warned: map[string]bool{},
	}
}

func (a *applier) Pull() (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull()
		if EOS(batch, err) {
			if s := a.expr.Warning(); s != "" {
				a.maybeWarn(s)
			}
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

func (a *applier) maybeWarn(s string) {
	if !a.warned[s] {
		a.pctx.Warnings <- fmt.Sprintf("%s: %s", a.expr, s)
		a.warned[s] = true
	}
}

func (a *applier) Done() {
	a.parent.Done()
}
