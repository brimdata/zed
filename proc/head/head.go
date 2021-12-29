package head

import (
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent       zbuf.Puller
	limit, count int
}

func New(parent zbuf.Puller, limit int) *Proc {
	return &Proc{
		parent: parent,
		limit:  limit,
	}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	if p.count >= p.limit {
		// If we are at limit we already sent a done upstream,
		// so for either sense of the done flag, we return EOS
		// and reset our state.
		p.count = 0
		return nil, nil
	}
	if done {
		b, err := p.parent.Pull(true)
		if err != nil {
			return nil, err
		}
		if b != nil {
			panic("non-nil done batch")
		}
		p.count = 0
		return nil, nil
	}
again:
	batch, err := p.parent.Pull(false)
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
	// Signal to the parent that we are done and set the done
	// flag so any downstream dones will not be erroneously
	// propagated since this path is already done.
	if _, err := p.parent.Pull(true); err != nil {
		return nil, err
	}
	p.count = p.limit
	return zbuf.NewBatch(batch, vals[:remaining]), nil
}
