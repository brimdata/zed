package data

import (
	"context"

	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
)

func LookupSeekRange(ctx context.Context, engine storage.Engine, path *storage.URI,
	o *Object, cmp expr.CompareFn, filter *expr.SpanFilter, countSpan extent.Span) (seekindex.Range, error) {
	r, err := engine.Get(ctx, o.SeekIndexURI(path))
	if err != nil {
		return seekindex.Range{}, err
	}
	defer r.Close()
	rg := seekindex.Range{Start: -1}
	reader := seekindex.NewSectionReader(r, o.Last, o.Count, o.Size, cmp)
	for {
		s, err := reader.Next()
		if s == nil || err != nil {
			return rg, err
		}
		if filter != nil && filter.Eval(s.Keys.First(), s.Keys.Last()) {
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
