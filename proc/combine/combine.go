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
	cancel   context.CancelFunc
	once     sync.Once
	ch       <-chan proc.Result
	parents  []*puller
	nparents int
}

type puller struct {
	proc.Interface
	ctx context.Context
	ch  chan<- proc.Result
}

func (p *puller) run() {
	for {
		batch, err := p.Pull()
		select {
		case p.ch <- proc.Result{batch, err}:
			if batch == nil || err != nil {
				return
			}
		case <-p.ctx.Done():
			p.Done()
			return
		}
	}
}

func New(pctx *proc.Context, parents []proc.Interface) *Proc {
	ch := make(chan proc.Result)
	ctx, cancel := context.WithCancel(pctx.Context)
	pullers := make([]*puller, 0, len(parents))
	for _, parent := range parents {
		pullers = append(pullers, &puller{parent, ctx, ch})
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
		for _, m := range p.parents {
			go m.run()
		}
	})
	for {
		if p.nparents == 0 {
			return nil, nil
		}
		select {
		case res := <-p.ch:
			if res.Batch != nil || res.Err != nil {
				return res.Batch, res.Err
			}
			p.nparents--
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		}
	}
}

func (m *Proc) Done() {
	m.cancel()
}
