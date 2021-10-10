package until

import (
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

// Proc computes aggregations using an Aggregator.
type Proc struct {
	pctx     *proc.Context
	parent   proc.Interface
	agg      *Aggregator
	once     sync.Once
	resultCh chan proc.Result
}

func New(pctx *proc.Context, parent proc.Interface, until expr.Filter, keys []expr.Assignment, aggNames field.List, aggs []*expr.Aggregator, limit int) (*Proc, error) {
	names := make(field.List, 0, len(keys)+len(aggNames))
	for _, e := range keys {
		names = append(names, e.LHS)
	}
	names = append(names, aggNames...)
	builder, err := zed.NewColumnBuilder(pctx.Zctx, names)
	if err != nil {
		return nil, err
	}
	valRefs := make([]expr.Evaluator, 0, len(aggNames))
	for _, fieldName := range aggNames {
		valRefs = append(valRefs, expr.NewDotExpr(fieldName))
	}
	keyRefs := make([]expr.Evaluator, 0, len(keys))
	keyExprs := make([]expr.Evaluator, 0, len(keys))
	for _, e := range keys {
		keyRefs = append(keyRefs, expr.NewDotExpr(e.LHS))
		keyExprs = append(keyExprs, e.RHS)
	}
	agg, err := NewAggregator(pctx.Zctx, until, keyRefs, keyExprs, valRefs, aggs, builder, limit)
	if err != nil {
		return nil, err
	}
	return &Proc{
		pctx:     pctx,
		parent:   parent,
		agg:      agg,
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

func (p *Proc) Done() {
	p.parent.Done()
}

func (p *Proc) run() {
	for {
		var out zbuf.Array
		batch, err := p.parent.Pull()
		if err != nil {
			p.shutdown(err)
			return
		}
		if batch == nil {
			p.sendResult(nil, err)
			return
		}
		for k := 0; k < batch.Length(); k++ {
			rec, err := p.agg.Consume(batch.Index(k))
			if err != nil {
				batch.Unref()
				p.shutdown(err)
				return
			}
			out.Append(rec)
		}
		batch.Unref()
		if out.Length() > 0 {
			p.sendResult(out, nil)
		}
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
	case <-p.pctx.Done():
	}
}

func (p *Proc) shutdown(err error) {
	// Make sure we cleanup before sending EOS.  Otherwise, the process
	// could exit before we remove the spill directory.
	p.sendResult(nil, err)
	close(p.resultCh)
}
