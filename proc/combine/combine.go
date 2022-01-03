// A combine proc merges multiple upstream inputs into one output.
package combine

import (
	"context"
	"sync"

	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	ctx      context.Context
	once     sync.Once
	resultCh <-chan result
	parents  []*puller
	blocked  map[*puller]struct{}
	err      error
}

type puller struct {
	proc.Interface
	ctx      context.Context
	resultCh chan<- result
	resumeCh chan struct{}
	doneCh   chan struct{}
}

func newPuller(ctx context.Context, parent proc.Interface, resultCh chan<- result) *puller {
	return &puller{
		Interface: proc.NewCatcher(parent),
		ctx:       ctx,
		resultCh:  resultCh,
		resumeCh:  make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

type result struct {
	batch  zbuf.Batch
	err    error
	puller *puller
}

func (p *puller) run() {
	for {
		batch, err := p.Pull()
		select {
		case p.resultCh <- result{batch, err, p}:
			if err != nil {
				return
			}
			if batch == nil {
				if !p.waitToResume() {
					return
				}
			}
		case <-p.doneCh:
			// Drop the pending batch and initiate a done...
			if !p.propagateDone() {
				return
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *puller) propagateDone() bool {
	p.Done()
	for {
		batch, err := p.Pull()
		if err != nil {
			p.resultCh <- result{nil, err, p}
			return false
		}
		if batch == nil {
			p.resultCh <- result{nil, nil, p}
			return p.waitToResume()
		}
		batch.Unref()
	}
}

func (p *puller) waitToResume() bool {
	select {
	case <-p.resumeCh:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func New(pctx *proc.Context, parents []proc.Interface) *Proc {
	resultCh := make(chan result)
	ctx := pctx.Context
	pullers := make([]*puller, 0, len(parents))
	for _, parent := range parents {
		pullers = append(pullers, newPuller(ctx, parent, resultCh))
	}
	return &Proc{
		ctx:      ctx,
		resultCh: resultCh,
		parents:  pullers,
		blocked:  make(map[*puller]struct{}),
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() {
		for _, m := range p.parents {
			go m.run()
		}
	})
	if p.err != nil {
		return nil, p.err
	}
	for {
		if len(p.blocked) == len(p.parents) {
			for _, parent := range p.parents {
				parent.resumeCh <- struct{}{}
				delete(p.blocked, parent)
			}
			return nil, nil
		}
		select {
		case res := <-p.resultCh:
			if err := res.err; err != nil {
				p.err = err
				return nil, err
			}
			if res.batch == nil {
				p.blocked[res.puller] = struct{}{}
				continue
			}
			return res.batch, nil
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		}
	}
}

func (p *Proc) Done() {
	for _, parent := range p.parents {
		if _, ok := p.blocked[parent]; !ok {
			parent.doneCh <- struct{}{}
		}
	}
}
