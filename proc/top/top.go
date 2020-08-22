package top

import (
	"container/heap"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

const defaultTopLimit = 100

// Top is similar to proc.Sort with a view key differences:
// - It only sorts in descending order.
// - It utilizes a MaxHeap, immediately discarding records that are not in
// the top N of the sort.
// - It has a hidden option (FlushEvery) to sort and emit on every batch.
type Proc struct {
	proc.Parent
	limit      int
	fields     []expr.FieldExprResolver
	records    *expr.RecordSlice
	compare    expr.CompareFn
	flushEvery bool
}

func New(parent proc.Interface, limit int, fields []expr.FieldExprResolver, flushEvery bool) *Proc {
	if limit == 0 {
		limit = defaultTopLimit
	}
	return &Proc{
		Parent:     proc.Parent{parent},
		limit:      limit,
		fields:     fields,
		flushEvery: flushEvery,
	}
}

func (t *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := t.Parent.Pull()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return t.sorted(), nil
		}
		for k := 0; k < batch.Length(); k++ {
			t.consume(batch.Index(k))
		}
		batch.Unref()
		if t.flushEvery {
			return t.sorted(), nil
		}
	}
}

func (t *Proc) consume(rec *zng.Record) {
	if t.fields == nil {
		fld := sort.GuessSortKey(rec)
		resolver := func(r *zng.Record) zng.Value {
			e, err := r.Access(fld)
			if err != nil {
				return zng.Value{}
			}
			return e
		}
		t.fields = []expr.FieldExprResolver{resolver}
	}
	if t.records == nil {
		t.compare = expr.NewCompareFn(false, t.fields...)
		t.records = expr.NewRecordSlice(t.compare)
		heap.Init(t.records)
	}
	if t.records.Len() < t.limit || t.compare(t.records.Index(0), rec) < 0 {
		heap.Push(t.records, rec.Keep())
	}
	if t.records.Len() > t.limit {
		heap.Pop(t.records)
	}
}

func (t *Proc) sorted() zbuf.Batch {
	if t.records == nil {
		return nil
	}
	out := make([]*zng.Record, t.records.Len())
	for i := t.records.Len() - 1; i >= 0; i-- {
		rec := heap.Pop(t.records).(*zng.Record)
		out[i] = rec
	}
	// clear records
	t.records = nil
	return zbuf.NewArray(out)
}
