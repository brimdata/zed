package top

import (
	"container/heap"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/runtime/op/sort"
	"github.com/brimdata/zed/zbuf"
)

const defaultTopLimit = 100

// Top is similar to op.Sort with a view key differences:
// - It only sorts in descending order.
// - It utilizes a MaxHeap, immediately discarding records that are not in
// the top N of the sort.
// - It has a hidden option (FlushEvery) to sort and emit on every batch.
type Proc struct {
	parent     zbuf.Puller
	zctx       *zed.Context
	limit      int
	fields     []expr.Evaluator
	records    *expr.RecordSlice
	compare    expr.CompareFn
	flushEvery bool
}

func New(parent zbuf.Puller, zctx *zed.Context, limit int, fields []expr.Evaluator, flushEvery bool) *Proc {
	if limit == 0 {
		limit = defaultTopLimit
	}
	return &Proc{
		parent:     parent,
		limit:      limit,
		fields:     fields,
		flushEvery: flushEvery,
	}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return p.sorted(), nil
		}
		vals := batch.Values()
		for i := range vals {
			p.consume(&vals[i])
		}
		batch.Unref()
		if p.flushEvery {
			return p.sorted(), nil
		}
	}
}

func (p *Proc) consume(rec *zed.Value) {
	if p.fields == nil {
		fld := sort.GuessSortKey(rec)
		accessor := expr.NewDottedExpr(p.zctx, fld)
		p.fields = []expr.Evaluator{accessor}
	}
	if p.records == nil {
		p.compare = expr.NewCompareFn(false, p.fields...)
		p.records = expr.NewRecordSlice(p.compare)
		heap.Init(p.records)
	}
	if p.records.Len() < p.limit || p.compare(p.records.Index(0), rec) < 0 {
		heap.Push(p.records, rec.Copy())
	}
	if p.records.Len() > p.limit {
		heap.Pop(p.records)
	}
}

func (t *Proc) sorted() zbuf.Batch {
	if t.records == nil {
		return nil
	}
	out := make([]zed.Value, t.records.Len())
	for i := t.records.Len() - 1; i >= 0; i-- {
		out[i] = *heap.Pop(t.records).(*zed.Value)
	}
	// clear records
	t.records = nil
	return zbuf.NewArray(out)
}
