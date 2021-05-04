// A mux proc merges multiple upstream inputs into one output like combine
// but labels each batch.
package mux

import (
	"context"
	"sync"

	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type Batch struct {
	zbuf.Batch
	Label int
}

type Proc struct {
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once
	ch       <-chan result
	parents  []*puller
	nparents int
}

type result struct {
	batch zbuf.Batch
	label int
	err   error
}

type puller struct {
	proc.Interface
	ctx   context.Context
	ch    chan<- result
	label int
}

func (p *puller) run() {
	for {
		batch, err := p.Pull()
		select {
		case p.ch <- result{batch, p.label, err}:
			if proc.EOS(batch, err) {
				return
			}
		case <-p.ctx.Done():
			p.Done()
			return
		}
	}
}

func New(pctx *proc.Context, parents []proc.Interface) *Proc {
	ch := make(chan result)
	ctx, cancel := context.WithCancel(pctx.Context)
	pullers := make([]*puller, 0, len(parents))
	for label, parent := range parents {
		pullers = append(pullers, &puller{parent, ctx, ch, label})
	}
	return &Proc{
		ctx:      ctx,
		cancel:   cancel,
		ch:       ch,
		parents:  pullers,
		nparents: len(parents),
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() {
		for _, puller := range p.parents {
			go puller.run()
		}
	})
	for {
		if p.nparents == 0 {
			return nil, nil
		}
		select {
		case res := <-p.ch:
			if proc.EOS(res.batch, res.err) {
				p.nparents--
			}
			return &Batch{res.batch, res.label}, res.err
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		}
	}
}

func (p *Proc) Done() {
	p.cancel()
}
