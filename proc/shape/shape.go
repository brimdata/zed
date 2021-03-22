package shape

import (
	"sync"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

var MemMaxBytes = 128 * 1024 * 1024
var BatchSize = 100

type Proc struct {
	pctx   *proc.Context
	parent proc.Interface

	shaper   *Shaper
	once     sync.Once
	resultCh chan proc.Result
}

func New(pctx *proc.Context, parent proc.Interface) (*Proc, error) {
	return &Proc{
		pctx:     pctx,
		parent:   parent,
		shaper:   NewShaper(pctx.Zctx, MemMaxBytes),
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
	err := p.pullInput()
	if err == nil {
		err = p.pushOutput()
	}
	p.shutdown(err)
}

func (p *Proc) pullInput() error {
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		batch, err := p.parent.Pull()
		if err != nil || batch == nil {
			return err
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
		if err := p.shaper.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proc) pushOutput() error {
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		batch, err := p.nextBatch()
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
	if err2 := p.shaper.Close(); err == nil {
		err = err2
	}
	p.sendResult(nil, err)
	close(p.resultCh)
}

func (p *Proc) Done() {
	p.parent.Done()
}

func (p *Proc) nextBatch() (zbuf.Batch, error) {
	var out []*zng.Record
	for len(out) < BatchSize {
		rec, err := p.shaper.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		out = append(out, rec)
	}
	if out == nil {
		return nil, nil
	}
	return zbuf.Array(out), nil
}
