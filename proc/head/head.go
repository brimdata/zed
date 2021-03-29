package head

import (
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
)

type Proc struct {
	parent       proc.Interface
	limit, count int
}

func New(parent proc.Interface, limit int) *Proc {
	return &Proc{
		parent: parent,
		limit:  limit,
	}
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	remaining := p.limit - p.count
	if remaining <= 0 {
		return nil, nil
	}
	batch, err := p.parent.Pull()
	if proc.EOS(batch, err) {
		return nil, err
	}
	n := batch.Length()
	if n < remaining {
		// This batch has fewer than the needed records.
		// Send them all downstream and update the count.
		p.count += n
		return batch, nil
	}
	defer batch.Unref()
	// This batch has more than the needed records.
	// Create a new batch and copy only the needed records.
	// Then signal to the upstream that we're done.
	recs := make([]*zng.Record, remaining)
	for k := 0; k < remaining; k++ {
		recs[k] = batch.Index(k).Keep()
	}
	p.count = p.limit
	p.Done()
	return zbuf.Array(recs), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}
