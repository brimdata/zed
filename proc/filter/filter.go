package filter

import (
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Proc struct {
	filter.Filter
	parent proc.Interface
}

func New(parent proc.Interface, f filter.Filter) *Proc {
	return &Proc{
		parent: parent,
		Filter: f,
	}
}

func (f *Proc) Pull() (zbuf.Batch, error) {
	batch, err := f.parent.Pull()
	if proc.EOS(batch, err) {
		return nil, err
	}
	defer batch.Unref()
	// Now we'll a new batch with the (sub)set of reords that match.
	out := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		r := batch.Index(k)
		if f.Filter(r) {
			out = append(out, r.Keep())
		}
	}
	return zbuf.Array(out), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}
