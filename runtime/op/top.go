package op

import (
	"container/heap"
	"slices"

	"github.com/superdb/super"
	"github.com/superdb/super/runtime/sam/expr"
	"github.com/superdb/super/runtime/sam/op/sort"
	"github.com/superdb/super/scode"
	"github.com/superdb/super/vector"
	"github.com/superdb/super/vector/vio"
)

type Top struct {
	sctx         *super.Context
	parent       vio.Puller
	limit        int
	exprs        []expr.SortExpr
	guessReverse bool

	builder scode.Builder
	compare expr.CompareFn
	eos     bool
	records *recordSlice
}

func NewTop(sctx *super.Context, parent vio.Puller, limit int, exprs []expr.SortExpr, guessReverse bool) *Top {
	return &Top{
		sctx:         sctx,
		parent:       parent,
		limit:        limit,
		exprs:        exprs,
		guessReverse: guessReverse,
	}
}

func (t *Top) Pull(done bool) (vector.Any, error) {
	if t.eos {
		t.eos = false
		return nil, nil
	}
	for {
		vec, err := t.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			if t.records == nil {
				return nil, nil
			}
			t.eos = true
			return t.sorted(), nil
		}
		for i := range vec.Len() {
			t.consume(vector.ValueAt(&t.builder, vec, i))
		}
	}
}

func (t *Top) consume(val super.Value) {
	if t.records == nil {
		if t.compare == nil {
			comparator := sort.NewComparator(t.sctx, t.exprs, val, t.guessReverse)
			// package heap implements a min-heap.  Invert the comparison result to get a max-heap.
			t.compare = func(a, b super.Value) int { return comparator.Compare(a, b) * -1 }
		}
		t.records = newRecordSlice(t.compare)
		heap.Init(t.records)
	}
	if t.records.Len() < t.limit || t.compare(t.records.Index(0), val) < 0 {
		heap.Push(t.records, val.Copy())
	}
	if t.records.Len() > t.limit {
		heap.Pop(t.records)
	}
}

func (t *Top) sorted() vector.Any {
	// Transform t.record.vals from a max-heap to an ascending list that we
	// can use to write values to a vector builder in ascending order.
	vals := t.records.vals
	for i := range slices.Backward(vals) {
		vals[i] = heap.Pop(t.records).(super.Value)
	}
	t.records = nil
	b := vector.NewDynamicValueBuilder()
	for _, val := range vals {
		b.Write(val)
	}
	return b.Build(t.sctx)
}

type recordSlice struct {
	vals    []super.Value
	compare expr.CompareFn
}

func newRecordSlice(compare expr.CompareFn) *recordSlice {
	return &recordSlice{compare: compare}
}

// Swap implements sort.Interface for *Record slices.
func (r *recordSlice) Len() int { return len(r.vals) }

// Swap implements sort.Interface for *Record slices.
func (r *recordSlice) Swap(i, j int) { r.vals[i], r.vals[j] = r.vals[j], r.vals[i] }

// Less implements sort.Interface for *Record slices.
func (r *recordSlice) Less(i, j int) bool {
	return r.compare(r.vals[i], r.vals[j]) < 0
}

// Push adds x as element Len(). Implements heap.Interface.
func (r *recordSlice) Push(rec any) {
	r.vals = append(r.vals, rec.(super.Value))
}

// Pop removes the first element in the array. Implements heap.Interface.
func (r *recordSlice) Pop() any {
	rec := r.vals[len(r.vals)-1]
	r.vals = r.vals[:len(r.vals)-1]
	return rec
}

// Index returns the ith record.
func (r *recordSlice) Index(i int) super.Value {
	return r.vals[i]
}
