package proc

import (
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Function interface {
	Apply(*zng.Record) (*zng.Record, error)
	Warning() string
}

type Applier struct {
	pctx     *Context
	parent   Interface
	function Function
}

func NewApplier(pctx *Context, parent Interface, f Function) *Applier {
	return &Applier{
		pctx:     pctx,
		parent:   parent,
		function: f,
	}
}

func (a *Applier) warn() {
	if s := a.function.Warning(); s != "" {
		a.pctx.Warnings <- s
	}
}

func (a *Applier) Pull() (zbuf.Batch, error) {
	for {
		batch, err := a.parent.Pull()
		if EOS(batch, err) {
			a.warn()
			return nil, err
		}
		// Make new records with only the fields specified.
		// If a field specified doesn't exist, we don't include that record.
		// If the types change for the fields specified, we drop those records.
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

func (a *Applier) Done() {
	a.parent.Done()
}
