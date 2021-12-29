package yield

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent zbuf.Puller
	exprs  []expr.Evaluator
}

func New(parent zbuf.Puller, exprs []expr.Evaluator) *Proc {
	return &Proc{
		parent: parent,
		exprs:  exprs,
	}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull(done)
		if batch == nil || err != nil {
			return nil, err
		}
		vals := batch.Values()
		recs := make([]zed.Value, 0, len(p.exprs)*len(vals))
		for i := range vals {
			for _, e := range p.exprs {
				out := e.Eval(batch, &vals[i])
				if out.IsMissing() {
					continue
				}
				// Copy is necessary because argument bytes
				// can be reused.
				recs = append(recs, *out.Copy())
			}
		}
		defer batch.Unref()
		if len(recs) > 0 {
			return zbuf.NewBatch(batch, recs), nil
		}
	}
}
