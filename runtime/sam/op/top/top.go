package top

import (
	"container/heap"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/op/sort"
	"github.com/brimdata/zed/zbuf"
)

const defaultTopLimit = 100

// Top is similar to op.Sort with a view key differences:
// - It only sorts in descending order.
// - It utilizes a MaxHeap, immediately discarding records that are not in
// the top N of the sort.
// - It has a hidden option (FlushEvery) to sort and emit on every batch.
type Op struct {
	zctx       *zed.Context
	parent     zbuf.Puller
	limit      int
	fields     []expr.Evaluator
	flushEvery bool
	batches    map[zed.Value]zbuf.Batch
	records    *expr.RecordSlice
	compare    expr.CompareFn
}

func New(zctx *zed.Context, parent zbuf.Puller, limit int, fields []expr.Evaluator, flushEvery bool) *Op {
	if limit == 0 {
		limit = defaultTopLimit
	}
	return &Op{
		zctx:       zctx,
		parent:     parent,
		limit:      limit,
		fields:     fields,
		flushEvery: flushEvery,
		batches:    make(map[zed.Value]zbuf.Batch),
	}
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := o.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return o.sorted(), nil
		}
		vals := batch.Values()
		for i := range vals {
			o.consume(batch, vals[i])
		}
		batch.Unref()
		if o.flushEvery {
			return o.sorted(), nil
		}
	}
}

func (o *Op) consume(batch zbuf.Batch, rec zed.Value) {
	if o.fields == nil {
		fld := sort.GuessSortKey(rec)
		accessor := expr.NewDottedExpr(o.zctx, fld)
		o.fields = []expr.Evaluator{accessor}
	}
	if o.records == nil {
		o.compare = expr.NewCompareFn(false, o.fields...)
		o.records = expr.NewRecordSlice(o.compare)
		heap.Init(o.records)
	}
	if o.records.Len() < o.limit || o.compare(o.records.Index(0), rec) < 0 {
		heap.Push(o.records, rec)
		if _, ok := rec.Arena(); ok {
			batch.Ref()
			o.batches[rec] = batch
		}
	}
	if o.records.Len() > o.limit {
		val := heap.Pop(o.records).(zed.Value)
		if batch, ok := o.batches[val]; ok {
			batch.Unref()
		}
	}
}

func (o *Op) sorted() zbuf.Batch {
	if o.records == nil {
		return nil
	}
	arena := zed.NewArena()
	defer arena.Unref()
	out := make([]zed.Value, o.records.Len())
	for i := o.records.Len() - 1; i >= 0; i-- {
		out[i] = heap.Pop(o.records).(zed.Value).Copy(arena)
	}
	// clear records
	o.records = nil
	return zbuf.NewArray(arena, out)
}
