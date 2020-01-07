package proc

import (
	"container/heap"

	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
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
	out        []*zbuf.Record
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

func (s *Top) Pull() (zbuf.Batch, error) {
	for {
		batch, err := s.Get()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return s.sorted(), nil
		}
		for k := 0; k < batch.Length(); k++ {
			s.consume(batch.Index(k))
		}
		batch.Unref()
		if s.flushEvery {
			return s.sorted(), nil
		}
	}
}

func (s *Top) consume(rec *zbuf.Record) {
	if s.fields == nil {
		fld := guessSortField(rec)
		resolver := func(r *zbuf.Record) zng.TypedEncoding {
			e, err := r.Access(fld)
			if err != nil {
				return zng.TypedEncoding{}
			}
			return e
		}
		s.fields = []expr.FieldExprResolver{resolver}
	}
	if s.records == nil {
		s.sorter = expr.NewSortFn(false, s.fields...)
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

func (s *Top) sorted() zbuf.Batch {
	if s.records == nil {
		return nil
	}
	out := make([]*zbuf.Record, s.records.Len())
	for i := s.records.Len() - 1; i >= 0; i-- {
		rec := heap.Pop(s.records).(*zbuf.Record)
		out[i] = rec
	}
	// clear records
	s.records = nil
	return zbuf.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}
