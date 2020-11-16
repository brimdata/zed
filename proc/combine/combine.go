// A combine proc merges multiple upstream inputs into one output.
package combine

import (
	"context"
	"sync"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
)

type Proc struct {
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once
	ch       <-chan proc.Result
	parents  []*runnerProc
	nparents int
}

type runnerProc struct {
	ctx    context.Context
	parent proc.Interface
	ch     chan<- proc.Result
	doneCh <-chan struct{}
}

func (r *runnerProc) run() {
	for {
		batch, err := r.parent.Pull()
		select {
		case r.ch <- proc.Result{batch, err}:
			if proc.EOS(batch, err) {
				return
			}
		case <-r.ctx.Done():
			r.parent.Done()
			return
		}
	}
}

func New(pctx *proc.Context, parents []proc.Interface) *Proc {
	ch := make(chan proc.Result)
	ctx, cancel := context.WithCancel(pctx.Context)
	runners := make([]*runnerProc, 0, len(parents))
	for _, parent := range parents {
		runners = append(runners, &runnerProc{
			ctx:    ctx,
			parent: parent,
			ch:     ch,
		})
	}
	return &Proc{
		ctx:      ctx,
		cancel:   cancel,
		ch:       ch,
		parents:  runners,
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
