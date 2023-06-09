package sort

import (
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/op/spill"
	"github.com/brimdata/zed/zbuf"
)

// MemMaxBytes specifies the maximum amount of memory that each sort proc
// will consume.
var MemMaxBytes = 128 * 1024 * 1024

type Op struct {
	octx       *op.Context
	parent     zbuf.Puller
	order      order.Which
	nullsFirst bool

	fieldResolvers []expr.Evaluator
	lastBatch      zbuf.Batch
	once           sync.Once
	resultCh       chan op.Result
	comparator     *expr.Comparator
}

func New(octx *op.Context, parent zbuf.Puller, fields []expr.Evaluator, order order.Which, nullsFirst bool) (*Op, error) {
	return &Op{
		octx:           octx,
		parent:         parent,
		order:          order,
		nullsFirst:     nullsFirst,
		fieldResolvers: fields,
		resultCh:       make(chan op.Result),
	}, nil
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	o.once.Do(func() {
		// Block p.ctx's cancel function until p.run finishes its
		// cleanup.
		o.octx.WaitGroup.Add(1)
		go o.run()
	})
	for {
		r, ok := <-o.resultCh
		if !ok {
			return nil, o.octx.Err()
		}
		if !done || r.Batch == nil || r.Err != nil {
			return r.Batch, r.Err
		}
		r.Batch.Unref()
	}
}

func (o *Op) run() {
	defer close(o.resultCh)
	var spiller *spill.MergeSort
	defer func() {
		if spiller != nil {
			spiller.Cleanup()
		}
		// Tell p.ctx's cancel function that we've finished our cleanup.
		o.octx.WaitGroup.Done()
	}()
	var nbytes int
	var out []zed.Value
	for {
		batch, err := o.parent.Pull(false)
		if err != nil {
			if ok := o.sendResult(nil, err); !ok {
				return
			}
			continue
		}
		if batch == nil {
			if spiller == nil {
				if len(out) > 0 {
					if ok := o.send(out); !ok {
						return
					}
				}
				if ok := o.sendResult(nil, nil); !ok {
					return
				}
				nbytes = 0
				out = nil
				continue
			}
			if len(out) > 0 {
				if err := spiller.Spill(o.octx.Context, out); err != nil {
					if ok := o.sendResult(nil, err); !ok {
						return
					}
					spiller = nil
					nbytes = 0
					out = nil
					continue
				}
			}
			if ok := o.sendSpills(spiller); !ok {
				return
			}
			spiller.Cleanup()
			spiller = nil
			nbytes = 0
			out = nil
			continue
		}
		// Safe because batch.Unref is never called.
		o.lastBatch = batch
		var delta int
		out, delta = o.append(out, batch)
		if o.comparator == nil && len(out) > 0 {
			o.setComparator(&out[0])
		}
		nbytes += delta
		if nbytes < MemMaxBytes {
			continue
		}
		if spiller == nil {
			spiller, err = spill.NewMergeSort(o.comparator)
			if err != nil {
				if ok := o.sendResult(nil, err); !ok {
					return
				}
				out = nil
				nbytes = 0
				continue
			}
		}
		if err := spiller.Spill(o.octx.Context, out); err != nil {
			if ok := o.sendResult(nil, err); !ok {
				return
			}
		}
		out = nil
		nbytes = 0
	}
}

// send sorts vals in memory and sends the result downstream.
func (o *Op) send(vals []zed.Value) bool {
	o.comparator.SortStable(vals)
	out := zbuf.NewBatch(o.lastBatch, vals)
	return o.sendResult(out, nil)
}

func (o *Op) sendSpills(spiller *spill.MergeSort) bool {
	puller := zbuf.NewPuller(spiller)
	for {
		if err := o.octx.Err(); err != nil {
			return false
		}
		// Reading from the spiller merges the spilt files.
		b, err := puller.Pull(false)
		if ok := o.sendResult(b, err); !ok {
			return false
		}
		if b == nil || err != nil {
			return true
		}
	}
}

func (o *Op) sendResult(b zbuf.Batch, err error) bool {
	select {
	case o.resultCh <- op.Result{Batch: b, Err: err}:
		return true
	case <-o.octx.Done():
		return false
	}
}

func (o *Op) append(out []zed.Value, batch zbuf.Batch) ([]zed.Value, int) {
	var nbytes int
	vals := batch.Values()
	for i := range vals {
		val := &vals[i]
		nbytes += len(val.Bytes())
		// We're keeping records owned by batch so don't call Unref.
		out = append(out, *val)
	}
	return out, nbytes
}

func (o *Op) setComparator(r *zed.Value) {
	resolvers := o.fieldResolvers
	if resolvers == nil {
		fld := GuessSortKey(r)
		resolver := expr.NewDottedExpr(o.octx.Zctx, fld)
		resolvers = []expr.Evaluator{resolver}
	}
	reverse := o.order == order.Desc
	nullsMax := !o.nullsFirst
	if reverse {
		nullsMax = !nullsMax
	}
	o.comparator = expr.NewComparator(nullsMax, reverse, resolvers...).WithMissingAsNull()
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
	return field.Path{"ts"}
}

func firstMatchingField(typ *zed.TypeRecord, pred func(id int) bool) field.Path {
	for _, f := range typ.Fields {
		if pred(f.Type.ID()) {
			return field.Path{f.Name}
		}
		if typ := zed.TypeRecordOf(f.Type); typ != nil {
			if ff := firstMatchingField(typ, pred); ff != nil {
				return append(field.Path{f.Name}, ff...)
			}
		}
	}
	return nil
}
