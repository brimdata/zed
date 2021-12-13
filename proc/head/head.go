package head

import (
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
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
	batch, err := p.parent.Pull()
	if proc.EOS(batch, err) {
		p.count = 0
		return nil, err
	}
	remaining := p.limit - p.count
	if remaining <= 0 {
		batch.Unref()
		return nil, nil
	}
	vals := batch.Values()
	if n := len(vals); n < remaining {
		// This batch has fewer than the needed records.
		// Send them all downstream and update the count.
		p.count += n
		return batch, nil
	}
	// This batch has more than the needed records.
	// Signal to the upstream that we're done.  Then
	// return a batch with only the needed records.
	p.Done()
	p.count = p.limit
	return zbuf.Array(vals[:remaining]), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}
