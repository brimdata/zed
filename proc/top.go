package proc

import (
	"container/heap"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

const defaultTopLimit = 100

// Top is similar to proc.Sort with a view key differences:
// - It only sorts in descending order.
// - It utilizes a MaxHeap, immediately discarding records that are not in
// the top N of the sort.
// - It has a hidden option (FlushEvery) to sort and emit on every batch.
type Top struct {
	Base
	limit      int
	fields     []expr.FieldExprResolver
	records    *expr.RecordSlice
	sorter     expr.SortFn
	flushEvery bool
}

func NewTop(c *Context, parent Proc, limit int, fields []expr.FieldExprResolver, flushEvery bool) *Top {
	if limit == 0 {
		limit = defaultTopLimit
	}
	return &Top{
		Base:       Base{Context: c, Parent: parent},
		limit:      limit,
		fields:     fields,
		flushEvery: flushEvery,
	}
}

func (t *Top) Pull() (zbuf.Batch, error) {
	for {
		batch, err := t.Get()
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

func (t *Top) consume(rec *zng.Record) {
	if t.fields == nil {
		fld := guessSortField(rec)
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
		t.sorter = expr.NewSortFn(false, t.fields...)
		t.records = expr.NewRecordSlice(t.sorter)
		heap.Init(t.records)
	}
	if t.records.Len() < t.limit || t.sorter(t.records.Index(0), rec) < 0 {
		heap.Push(t.records, rec.Keep())
	}
	if t.records.Len() > t.limit {
		heap.Pop(t.records)
	}
}

func (t *Top) sorted() zbuf.Batch {
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
