package join

import (
	"context"

	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
)

type puller struct {
	zio.Reader
	op  zbuf.Puller
	ctx context.Context
	ch  chan op.Result
}

func newPuller(p zbuf.Puller, ctx context.Context) *puller {
	puller := &puller{
		op:  p,
		ctx: ctx,
		ch:  make(chan op.Result),
	}
	puller.Reader = zbuf.PullerReader(puller)
	return puller
}

func (p *puller) run() {
	for {
		batch, err := p.op.Pull(false)
		select {
		case p.ch <- op.Result{Batch: batch, Err: err}:
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
