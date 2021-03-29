package filter

import (
	"github.com/brimdata/zq/expr"
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
)

type Proc struct {
	expr.Filter
	parent proc.Interface
}

func New(parent proc.Interface, f expr.Filter) *Proc {
	return &Proc{
		parent: parent,
		Filter: f,
	}
}

func (f *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := f.parent.Pull()
		if proc.EOS(batch, err) {
			return nil, err
		}
		// Create a new batch containing matching records.
		out := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			r := batch.Index(k)
			if f.Filter(r) {
				out = append(out, r.Keep())
			}
		}
		batch.Unref()
		if len(out) > 0 {
			return zbuf.Array(out), nil
		}
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}
