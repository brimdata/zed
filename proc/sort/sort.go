package sort

import (
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/spill"
	"github.com/brimdata/zed/zbuf"
)

// MemMaxBytes specifies the maximum amount of memory that each sort proc
// will consume.
var MemMaxBytes = 128 * 1024 * 1024

type Proc struct {
	pctx       *proc.Context
	parent     zbuf.Puller
	order      order.Which
	nullsFirst bool

	fieldResolvers []expr.Evaluator
	once           sync.Once
	resultCh       chan proc.Result
	compareFn      expr.CompareFn
	ectx           expr.Context
	eof            bool
}

func New(pctx *proc.Context, parent zbuf.Puller, fields []expr.Evaluator, order order.Which, nullsFirst bool) (*Proc, error) {
	return &Proc{
		pctx:           pctx,
		parent:         parent,
		order:          order,
		nullsFirst:     nullsFirst,
		fieldResolvers: fields,
		resultCh:       make(chan proc.Result),
	}, nil
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	p.once.Do(func() { go p.run() })
	if done {
		for {
			r, ok := <-p.resultCh
			if !ok {
				return nil, p.pctx.Err()
			}
			if r.Batch == nil || r.Err != nil {
				return nil, r.Err
			}
		}
	}
	if r, ok := <-p.resultCh; ok {
		return r.Batch, r.Err
	}
	return nil, p.pctx.Err()
}

func (p *Proc) run() {
	defer close(p.resultCh)
	var spiller *spill.MergeSort
	defer func() {
		if spiller != nil {
			spiller.Cleanup()
		}
	}()
	var nbytes int
	var out []zed.Value
	for {
		batch, err := p.parent.Pull(false)
		if err != nil {
			if ok := p.sendResult(nil, err); !ok {
				return
			}
			continue
		}
		if batch == nil {
			if spiller == nil {
				if len(out) > 0 {
					if ok := p.send(out); !ok {
						return
					}
				}
				if ok := p.sendResult(nil, nil); !ok {
					return
				}
				nbytes = 0
				out = nil
				continue
			}
			if len(out) > 0 {
				if err := spiller.Spill(p.pctx.Context, out); err != nil {
					if ok := p.sendResult(nil, err); !ok {
						return
					}
					spiller = nil
					nbytes = 0
					out = nil
					continue
				}
			}
			if ok := p.sendSpills(spiller); !ok {
				return
			}
			spiller.Cleanup()
			spiller = nil
			nbytes = 0
			out = nil
			continue
		}
		var delta int
		out, delta = p.append(out, batch)
		if p.compareFn == nil && len(out) > 0 {
			p.setCompareFn(&out[0])
		}
		nbytes += delta
		if nbytes < MemMaxBytes {
			continue
		}
		if spiller == nil {
			spiller, err = spill.NewMergeSort(p.compareFn)
			if err != nil {
				if ok := p.sendResult(nil, err); !ok {
					return
				}
				out = nil
				nbytes = 0
				continue
			}
		}
		if err := spiller.Spill(p.pctx.Context, out); err != nil {
			if ok := p.sendResult(nil, err); !ok {
				return
			}
		}
		out = nil
		nbytes = 0
	}
}

// send sorts vals in memory and sends the result downstream.
func (p *Proc) send(vals []zed.Value) bool {
	expr.SortStable(vals, p.compareFn)
	//XXX bug: we need upstream ectx. See #3367
	array := zbuf.NewArray(vals)
	return p.sendResult(array, nil)
}

func (p *Proc) sendSpills(spiller *spill.MergeSort) bool {
	puller := zbuf.NewPuller(spiller, 100)
	for {
		if err := p.pctx.Err(); err != nil {
			return false
		}
		// Reading from the spiller merges the spilt files.
		b, err := puller.Pull(false)
		if ok := p.sendResult(b, err); !ok {
			return false
		}
		if b == nil || err != nil {
			return true
		}
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) bool {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
		return true
	case <-p.pctx.Done():
		return false
	}
}

func (p *Proc) append(out []zed.Value, batch zbuf.Batch) ([]zed.Value, int) {
	var nbytes int
	vals := batch.Values()
	for i := range vals {
		val := &vals[i]
		nbytes += len(val.Bytes)
		// We're keeping records owned by batch so don't call Unref.
		out = append(out, *val)
	}
	return out, nbytes
}

func (p *Proc) setCompareFn(r *zed.Value) {
	resolvers := p.fieldResolvers
	if resolvers == nil {
		fld := GuessSortKey(r)
		resolver := expr.NewDottedExpr(fld)
		resolvers = []expr.Evaluator{resolver}
	}
	nullsMax := !p.nullsFirst
	if p.order == order.Desc {
		nullsMax = !nullsMax
	}
	compareFn := expr.NewCompareFn(nullsMax, resolvers...)
	if p.order == order.Desc {
		p.compareFn = func(a, b *zed.Value) int { return compareFn(b, a) }
	} else {
		p.compareFn = compareFn
	}
}

func GuessSortKey(val *zed.Value) field.Path {
	recType := zed.TypeRecordOf(val.Type)
	if recType == nil {
		// A nil field.Path is equivalent to "this".
		return nil
	}
	if f := firstMatchingField(recType, zed.IsInteger); f != nil {
		return f
	}
	if f := firstMatchingField(recType, zed.IsFloat); f != nil {
		return f
	}
	isNotTime := func(id int) bool { return id != zed.IDTime }
	if f := firstMatchingField(recType, isNotTime); f != nil {
		return f
	}
	return field.New("ts")
}

func firstMatchingField(typ *zed.TypeRecord, pred func(id int) bool) field.Path {
	for _, col := range typ.Columns {
		if pred(col.Type.ID()) {
			return field.New(col.Name)
		}
		if typ := zed.TypeRecordOf(col.Type); typ != nil {
			if f := firstMatchingField(typ, pred); f != nil {
				return append(field.New(col.Name), f...)
			}
		}
	}
	return nil
}
