package proc

import (
	"fmt"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
)

type Function interface {
	fmt.Stringer
	Apply(*zng.Record) (*zng.Record, error)
	Warning() string
}

type applier struct {
	pctx     *Context
	parent   Interface
	function Function
	warned   map[string]bool
}

func FromFunction(pctx *Context, parent Interface, f Function) *applier {
	return &applier{
		pctx:     pctx,
		parent:   parent,
		function: f,
		warned:   map[string]bool{},
	}
}

func (a *applier) Pull() (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull()
		if EOS(batch, err) {
			if s := a.function.Warning(); s != "" {
				a.maybeWarn(s)
			}
			return nil, err
		}
		recs := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			in := batch.Index(k)
			out, err := a.function.Apply(in)
			if err != nil {
				a.maybeWarn(err.Error())
				continue
			}
			if out != nil {
				// Keep is necessary because Apply can return
				// its argument.
				recs = append(recs, out.Keep())
			}
		}
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.Array(recs), nil
		}
	}
}

func (a *applier) maybeWarn(s string) {
	if !a.warned[s] {
		a.pctx.Warnings <- fmt.Sprintf("%s: %s", a.function, s)
		a.warned[s] = true
	}
}

func (a *applier) Done() {
	a.parent.Done()
}
