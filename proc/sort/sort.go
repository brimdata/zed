package sort

import (
	"fmt"
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
	parent     proc.Interface
	order      order.Which
	nullsFirst bool

	fieldResolvers     []expr.Evaluator
	once               sync.Once
	resultCh           chan proc.Result
	compareFn          expr.CompareFn
	unseenFieldTracker *unseenFieldTracker
	ectx               expr.Context
	eof                bool
}

func New(pctx *proc.Context, parent proc.Interface, fields []expr.Evaluator, order order.Which, nullsFirst bool) (*Proc, error) {
	return &Proc{
		pctx:               pctx,
		parent:             parent,
		order:              order,
		nullsFirst:         nullsFirst,
		fieldResolvers:     fields,
		resultCh:           make(chan proc.Result),
		unseenFieldTracker: newUnseenFieldTracker(fields),
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
	defer close(p.resultCh)
	var spiller *spill.MergeSort
	var eof bool
	var nbytes int
	var out []zed.Value
	for {
		batch, err := p.parent.Pull()
		if err != nil {
			p.sendResult(nil, err)
			return
		}
		if batch == nil {
			if eof {
				if warnings := p.warnings(); warnings != nil {
					p.sendResult(warnings, nil)
				}
				return
			}
			eof = true
			if spiller == nil {
				if len(out) > 0 {
					p.send(out)
				}
				nbytes = 0
				out = nil
				continue
			}
			if len(out) > 0 {
				if err := spiller.Spill(p.pctx.Context, out); err != nil {
					spiller.Cleanup()
					p.sendResult(nil, err)
					return
				}
			}
			if err := p.sendSpills(spiller); err != nil {
				p.sendResult(nil, err)
				return
			}
			nbytes = 0
			out = nil
			spiller = nil
			continue
		}
		eof = false
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
				p.sendResult(nil, err)
				return
			}
		}
		if err := spiller.Spill(p.pctx.Context, out); err != nil {
			spiller.Cleanup()
			p.sendResult(nil, err)
			return
		}
		out = nil
		nbytes = 0
	}
}

// send sorts vals in memory and sends the result downstream.
func (p *Proc) send(vals []zed.Value) {
	expr.SortStable(vals, p.compareFn)
	//XXX bug: we need upstream ectx. See #3367
	array := zbuf.NewArray(vals)
	p.sendResult(array, nil)
}

func (p *Proc) sendSpills(spiller *spill.MergeSort) error {
	defer spiller.Cleanup()
	puller := zbuf.NewPuller(spiller, 100)
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		// Reading from the spiller merges the spilt files.
		b, err := puller.Pull()
		if b == nil || err != nil {
			return err
		}
		p.sendResult(b, nil)
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
	case <-p.pctx.Done():
	}
}

func (p *Proc) append(out []zed.Value, batch zbuf.Batch) ([]zed.Value, int) {
	var nbytes int
	ectx := batch.Context()
	vals := batch.Values()
	for i := range vals {
		val := &vals[i]
		p.unseenFieldTracker.update(ectx, val)
		nbytes += len(val.Bytes)
		// We're keeping records owned by batch so don't call Unref.
		out = append(out, *val)
	}
	return out, nbytes
}

func (p *Proc) warnings() *zbuf.Array {
	unseen := p.unseenFieldTracker.unseen()
	if len(unseen) == 0 {
		return nil
	}
	vals := make([]zed.Value, 0, len(unseen))
	for _, f := range unseen {
		name, _ := expr.DotExprToString(f)
		e := fmt.Sprintf("warning: sort field %q not present in input", name)
		vals = append(vals, *zed.NewValue(zed.TypeError, []byte(e)))
	}
	return zbuf.NewArray(vals)
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
