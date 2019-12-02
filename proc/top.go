package proc

import (
	"container/heap"

	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
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
	out        []*zson.Record
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

func (s *Top) Pull() (zson.Batch, error) {
	for {
		batch, err := s.Get()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return s.sorted(), nil
		}
		defer batch.Unref()
		for k := 0; k < batch.Length(); k++ {
			s.consume(batch.Index(k))
		}
		if s.flushEvery {
			return s.sorted(), nil
		}
	}
}

func (s *Top) consume(rec *zson.Record) {
	if s.fields == nil {
		fld := guessSortField(rec)
		resolver := func(r *zson.Record) zeek.TypedEncoding {
			e, err := r.Access(fld)
			if err != nil {
				return zeek.TypedEncoding{}
			}
			return e
		}
		s.fields = []expr.FieldExprResolver{resolver}
	}
	if s.records == nil {
		// 1 == MaxHeap
		s.sorter = expr.NewSortFn(1, s.fields...)
		s.records = expr.NewRecordSlice(s.sorter)
		heap.Init(s.records)
	}
	if s.records.Len() < s.limit || s.sorter(s.records.Index(0), rec) < 0 {
		heap.Push(s.records, rec.Keep())
	}
	if s.records.Len() > s.limit {
		heap.Pop(s.records)
	}
}

func (s *Top) sorted() zson.Batch {
	if s.records == nil {
		return nil
	}
	out := make([]*zson.Record, s.records.Len())
	for i := s.records.Len() - 1; i >= 0; i-- {
		rec := heap.Pop(s.records).(*zson.Record)
		out[i] = rec
	}
	// clear records
	s.records = nil
	return zson.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}
