package top

import (
	"container/heap"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/sort"
	"github.com/brimdata/zed/zbuf"
)

const defaultTopLimit = 100

// Top is similar to proc.Sort with a view key differences:
// - It only sorts in descending order.
// - It utilizes a MaxHeap, immediately discarding records that are not in
// the top N of the sort.
// - It has a hidden option (FlushEvery) to sort and emit on every batch.
type Proc struct {
	parent     proc.Interface
	limit      int
	fields     []expr.Evaluator
	records    *expr.RecordSlice
	compare    expr.CompareFn
	flushEvery bool
}

func New(parent proc.Interface, limit int, fields []expr.Evaluator, flushEvery bool) *Proc {
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

func (p *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return p.sorted(), nil
		}
		zvals := batch.Values()
		for i := range zvals {
			p.consume(&zvals[i])
		}
		batch.Unref()
		if p.flushEvery {
			return p.sorted(), nil
		}
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}

func (p *Proc) consume(rec *zed.Value) {
	if p.fields == nil {
		fld := sort.GuessSortKey(rec)
		accessor := expr.NewDotExpr(fld)
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
	return zbuf.Array(out)
}
