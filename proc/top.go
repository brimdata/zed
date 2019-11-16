package proc

import (
	"container/heap"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
)

const defaultTopLimit = 100

// TopProc is similar to proc.SortProc with a view key differences:
// - It only sorts in descending order.
// - It utilizes a MaxHeap, immediately discarding records are not in the top
// N of the sort.
// - It has a hidden option (FlushEvery) to sort and emit on every batch.
type TopProc struct {
	Base
	limit      int
	fields     []string
	sorter     *zson.Sorter
	flushEvery bool
	out        []*zson.Record
}

func NewTopProc(c *Context, parent Proc, limit int, fields []string, flushEvery bool) *TopProc {
	if limit == 0 {
		limit = defaultTopLimit
	}
	return &TopProc{
		Base:       Base{Context: c, Parent: parent},
		limit:      limit,
		fields:     fields,
		flushEvery: flushEvery,
	}
}

func (s *TopProc) Pull() (zson.Batch, error) {
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

func (s *TopProc) consume(rec *zson.Record) {
	if s.fields == nil {
		s.fields = []string{guessSortField(rec)}
	}
	if s.sorter == nil {
		// 1 == MaxHeap
		s.sorter = zson.NewSorter(1, s.fields...)
		heap.Init(s.sorter)
	}
	// XXX should probably compare the top element here instead.
	heap.Push(s.sorter, rec)
	if s.sorter.Len() > s.limit {
		heap.Pop(s.sorter)
	}
}

func (s *TopProc) sorted() zson.Batch {
	if s.sorter == nil {
		return nil
	}
	out := make([]*zson.Record, s.sorter.Len())
	for i := s.sorter.Len() - 1; i >= 0; i-- {
		rec := heap.Pop(s.sorter).(*zson.Record)
		out[i] = rec.Keep()
	}
	// clear sorter
	s.sorter = nil
	return zson.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}
