package fuse

import (
	"sync"

	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

var MemMaxBytes = 128 * 1024 * 1024
var BatchSize = 100

type Proc struct {
	pctx   *proc.Context
	parent proc.Interface

	fuser    *Fuser
	once     sync.Once
	resultCh chan proc.Result
}

func New(pctx *proc.Context, parent proc.Interface) (*Proc, error) {
	return &Proc{
		pctx:     pctx,
		parent:   parent,
		fuser:    NewFuser(pctx.Zctx, MemMaxBytes),
		resultCh: make(chan proc.Result),
	}, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() { go p.run() })
	if r, ok := <-p.resultCh; ok {
		return r.Batch, r.Err
	}
	return nil, p.pctx.Err()
}

func (p *Proc) run() {
	if err := p.pullInput(); err != nil {
		p.shutdown(err)
		return
	}
	p.shutdown(p.pushOutput())
}

func (p *Proc) pullInput() error {
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		batch, err := p.parent.Pull()
		if err != nil {
			return err
		}
		if batch == nil {
			return nil
		}
		if err := p.writeBatch(batch); err != nil {
			return err
		}
	}
}

func (p *Proc) writeBatch(batch zbuf.Batch) error {
	defer batch.Unref()
	l := batch.Length()
	for i := 0; i < l; i++ {
		rec := batch.Index(i)
		if err := p.fuser.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proc) pushOutput() error {
	puller := zbuf.NewPuller(p.fuser, BatchSize)
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		batch, err := puller.Pull()
		if err != nil || batch == nil {
			return err
		}
		p.sendResult(batch, nil)
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
	case <-p.pctx.Done():
	}
}

func (p *Proc) shutdown(err error) {
	if err2 := p.fuser.Close(); err == nil {
		err = err2
	}
	p.sendResult(nil, err)
	close(p.resultCh)
}

func (p *Proc) Done() {
	p.parent.Done()
}
