package proc

import (
	"fmt"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
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
}

func FromFunction(pctx *Context, parent Interface, f Function) *applier {
	return &applier{
		pctx:     pctx,
		parent:   parent,
		function: f,
	}
}

func (a *applier) warn() {
	if s := a.function.Warning(); s != "" {
		a.pctx.Warnings <- fmt.Sprintf("%s: %s", a.function, s)
	}
}

func (a *applier) Pull() (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull()
		if EOS(batch, err) {
			a.warn()
			return nil, err
		}
		recs := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			in := batch.Index(k)

			out, err := a.function.Apply(in)
			if err != nil {
				return nil, err
			}

			if out != nil {
				recs = append(recs, out)
			}
		}
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.Array(recs), nil
		}
	}
}

func (a *applier) Done() {
	a.parent.Done()
}
