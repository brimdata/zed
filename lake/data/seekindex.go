package data

import (
	"context"

	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
)

func LookupSeekRange(ctx context.Context, engine storage.Engine, path *storage.URI,
	obj *Object, cmp expr.CompareFn, filter *expr.SpanFilter, countSpan extent.Span, o order.Which) (*seekindex.Range, error) {
	r, err := engine.Get(ctx, obj.SeekIndexURI(path))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	rg := &seekindex.Range{Start: -1}
	reader := seekindex.NewSectionReader(r, obj.Last, obj.Count, obj.Size, cmp)
	swapper := expr.NewValueCompareFn(order.Asc, o == order.Asc)
	for {
		s, err := reader.Next()
		if s == nil || err != nil {
			return rg, err
		}
		first := s.Keys.First()
		last := s.Keys.Last()
		if swapper(first, last) > 0 {
			first, last = last, first
		}
		if filter != nil && filter.Eval(first, last) {
			continue
		}
		if countSpan != nil && !countSpan.Overlaps(s.Counts.First(), s.Counts.Last()) {
			continue
		}
		if rg.Start == -1 {
			rg.Start = s.Range.Start
		}
		rg.End = s.Range.End
	}
}
