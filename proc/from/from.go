package from

import (
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	sched  proc.Scheduler
	puller zbuf.PullerCloser
	done   bool
	err    error
}

func NewScheduler(pctx *proc.Context, sched proc.Scheduler) *Proc {
	return &Proc{
		sched: sched,
	}
}

func NewPuller(pctx *proc.Context, puller zbuf.PullerCloser) *Proc {
	return &Proc{
		puller: puller,
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (p *Proc) Pull() (zbuf.Batch, error) {
	if p.done {
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
		batch, err := p.puller.Pull()
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

func (p *Proc) Done() {
	if p.puller != nil {
		p.close(p.puller.Close())
	}
}
