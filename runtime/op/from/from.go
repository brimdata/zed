package from

import (
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	sched  op.Scheduler
	puller zbuf.PullerCloser
	done   bool
	err    error
}

func NewScheduler(pctx *op.Context, sched op.Scheduler) *Proc {
	return &Proc{
		sched: sched,
	}
}

func NewPuller(pctx *op.Context, puller zbuf.PullerCloser) *Proc {
	return &Proc{
		puller: puller,
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	if p.done {
		return nil, p.err
	}
	if done {
		if p.puller != nil {
			p.close(p.puller.Close())
			p.puller = nil
		}
		return nil, p.err
	}
	for {
		if p.puller == nil {
			if p.sched == nil {
				p.close(nil)
				return nil, nil
			}
			puller, err := p.sched.PullScanTask()
			if puller == nil || err != nil {
				p.close(err)
				return nil, err
			}
			p.puller = puller
		}
		batch, err := p.puller.Pull(false)
		if err != nil {
			p.close(err)
			p.puller.Close()
			return nil, err
		}
		if batch != nil {
			return batch, nil
		}
		if err := p.puller.Close(); err != nil {
			p.close(err)
			return nil, err
		}
		p.puller = nil
	}
}

func (p *Proc) close(err error) {
	p.err = err
	p.done = true
}
