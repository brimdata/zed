package traverse

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

var MemMaxBytes = 128 * 1024 * 1024

type Proc struct {
	SubProcSource *SubProcSource

	ctx     context.Context
	field   expr.Evaluator
	once    sync.Once
	parent  proc.Interface
	results chan proc.Result
	subProc proc.Interface
}

func New(ctx context.Context, parent proc.Interface, field expr.Evaluator) *Proc {
	return &Proc{
		// SubProcSource.ch needs to be buffered of len 2 so it will accept one
		// Batch and one EOS without blocking the run routine.
		SubProcSource: &SubProcSource{ctx: ctx, ch: make(chan zbuf.Batch, 2)},
		ctx:           ctx,
		field:         field,
		parent:        parent,
		results:       make(chan proc.Result),
	}
}

func (p *Proc) SetSubProc(subProc proc.Interface) {
	p.subProc = subProc
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() { go p.run() })
	if result, ok := <-p.results; ok {
		return result.Batch, result.Err
	}
	return nil, p.ctx.Err()
}

func (p *Proc) Done() { p.parent.Done() }

func (p *Proc) run() {
	var out zbuf.Array
	for ; p.ctx.Err() == nil; out = out[:0] {
		batch, err := p.parent.Pull()
		if proc.EOS(batch, err) {
			p.sendResult(nil, err)
			continue
		}
		for _, zval := range batch.Values() {
			zv, err := p.field.Eval(&zval)
			if err != nil {
				p.sendResult(nil, err)
				return
			}
			zvals, err := p.processValue(&zv)
			if err != nil {
				p.sendResult(nil, err)
				return
			}
			out = append(out, zvals...)
		}
		batch.Unref()
		p.sendResult(out, nil)
	}
}

func (p *Proc) processValue(zv *zed.Value) ([]zed.Value, error) {
	typ := zed.InnerType(zv.Type)
	if typ == nil {
		// XXX Support records and maps.
		zerr := zed.NewErrorf("value must be of type array or set, got: %s", zv.Type)
		return []zed.Value{zerr}, nil
	}
	iter := zcode.Iter(zv.Bytes)
	var zvals []zed.Value
	for {
		b, _, err := iter.Next()
		if b == nil {
			break
		}
		if err != nil {
			return nil, err
		}
		zvals = append(zvals, zed.Value{Type: typ, Bytes: b})
	}
	if p.subProc == nil {
		return zvals, nil
	}
	p.sendSubProcBatch(zvals)
	zvals = zvals[:0]
	for {
		batch, err := p.subProc.Pull()
		if proc.EOS(batch, err) {
			// Procs that can end early (head) may not receive all sent batches
			// so drain SubProcSource.ch so we don't have batches stuck in the
			// channel and causing weirdness on the next value.
			p.SubProcSource.drain()
			return zvals, err
		}
		zvals = append(zvals, batch.Values()...)
	}

}

func (p *Proc) sendResult(batch zbuf.Batch, err error) {
	select {
	case p.results <- proc.Result{Batch: batch, Err: err}:
	case <-p.ctx.Done():
	}
}

func (p *Proc) sendSubProcBatch(b zbuf.Array) {
	// Always send the batch followed by EOS which signals to procs in the
	// sub-sequence to reset state.
	for _, batch := range []zbuf.Batch{b, nil} {
		select {
		case p.SubProcSource.ch <- batch:
		case <-p.ctx.Done():
		}
	}
}

type SubProcSource struct {
	ctx context.Context
	ch  chan zbuf.Batch
}

func (s *SubProcSource) Pull() (zbuf.Batch, error) {
	var batch zbuf.Batch
	select {
	case batch = <-s.ch:
	case <-s.ctx.Done():
	}
	return batch, nil
}

func (s *SubProcSource) drain() {
	for {
		select {
		case <-s.ch:
		default:
			return
		}
	}
}

// Can be ignored.
func (s *SubProcSource) Done() {}
