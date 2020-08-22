package filter

import (
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Proc struct {
	proc.Parent
	filter.Filter
}

func New(parent proc.Parent, f filter.Filter) *Proc {
	return &Proc{
		Parent: parent,
		Filter: f,
	}
}

func (f *Proc) Pull() (zbuf.Batch, error) {
	batch, err := f.Get()
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
	return zbuf.NewArray(out), nil
}
