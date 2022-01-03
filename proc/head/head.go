package head

import (
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent       proc.Interface
	limit, count int
	err          error
	done         bool
}

func New(parent proc.Interface, limit int) *Proc {
	return &Proc{
		parent: parent,
		limit:  limit,
	}
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.done {
		p.done = false
		p.count = 0
		return nil, nil
	}
again:
	batch, err := p.parent.Pull()
	if batch == nil || err != nil {
		p.count = 0
		return nil, err
	}
	remaining := p.limit - p.count
	if remaining <= 0 {
		batch.Unref()
		goto again
	}
	vals := batch.Values()
	if n := len(vals); n < remaining {
		// This batch has fewer than the needed records.
		// Send them all downstream and update the count.
		p.count += n
		return batch, nil
	}
	// This batch has more than the needed records.
	// Signal to the upstream that we're done then we will resume
	// pulling until we hit eof.
	p.parent.Done()
	p.count = p.limit
	return zbuf.NewArray(vals[:remaining]), nil
}

func (p *Proc) Done() {
	// If we get a done from downstream, pull until EOS then start over.
	p.parent.Done()
	for {
		batch, err := p.parent.Pull()
		if err != nil {
			p.err = err
			return
		}
		if batch == nil {
			p.done = true
			return
		}
		batch.Unref()
	}
}
