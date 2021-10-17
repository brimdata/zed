package join

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type puller struct {
	proc  proc.Interface
	ctx   context.Context
	ch    chan proc.Result
	recs  []*zed.Value
	batch zbuf.Batch
	off   int
	len   int
}

func newPuller(p proc.Interface, ctx context.Context) *puller {
	return &puller{
		proc: p,
		ctx:  ctx,
		ch:   make(chan proc.Result),
	}
}

func (p *puller) run() {
	for {
		batch, err := p.proc.Pull()
		select {
		case p.ch <- proc.Result{batch, err}:
			if proc.EOS(batch, err) {
				close(p.ch)
				return
			}
		case <-p.ctx.Done():
			p.proc.Done()
			return
		}
	}
}

func (p *puller) Pull() (zbuf.Batch, error) {
	select {
	case res := <-p.ch:
		return res.Batch, res.Err
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

func (p *puller) Read() (*zed.Value, error) {
	if p.off >= p.len {
		// XXX last batch at EOS gets sent to GC
		if p.batch != nil {
			p.batch.Unref()
		}
		var err error
		p.batch, err = p.Pull()
		if p.batch == nil || err != nil {
			p.batch = nil
			return nil, err
		}
		p.off = 0
		p.len = p.batch.Length()
	}
	rec := p.batch.Index(p.off)
	p.off++
	return rec, nil
}

func (p *puller) Done() {
	p.proc.Done()
}
