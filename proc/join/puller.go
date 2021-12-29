package join

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type puller struct {
	proc  zbuf.Puller
	ctx   context.Context
	ch    chan proc.Result
	batch zbuf.Batch
	vals  []zed.Value
}

func newPuller(p zbuf.Puller, ctx context.Context) *puller {
	return &puller{
		proc: p,
		ctx:  ctx,
		ch:   make(chan proc.Result),
	}
}

func (p *puller) run() {
	for {
		batch, err := p.proc.Pull(false)
		select {
		case p.ch <- proc.Result{batch, err}:
			if batch == nil || err != nil {
				close(p.ch)
				return
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *puller) Pull(done bool) (zbuf.Batch, error) {
	select {
	case res := <-p.ch:
		return res.Batch, res.Err
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

func (p *puller) Read() (*zed.Value, error) {
	for len(p.vals) == 0 {
		if p.batch != nil {
			p.batch.Unref()
		}
		var err error
		p.batch, err = p.Pull(false)
		if p.batch == nil || err != nil {
			p.batch = nil
			return nil, err
		}
		p.vals = p.batch.Values()
	}
	rec := &p.vals[0]
	p.vals = p.vals[1:]
	return rec, nil
}
