package lake

import (
	"context"
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
)

func newSortedScanner(s *Scheduler, part Partition) (zbuf.Puller, error) {
	pullers := make([]zbuf.Puller, 0, len(part.Objects))
	pullersDone := func() {
		for _, p := range pullers {
			p.Pull(true)
		}
	}
	for _, o := range part.Objects {
		rg, err := s.rangeFinder(s.ctx, o)
		if err != nil {
			return nil, err
		}
		rc, err := o.NewReader(s.ctx, s.pool.engine, s.pool.DataPath, rg)
		if err != nil {
			pullersDone()
			return nil, err
		}
		scanner, err := zngio.NewReader(rc, s.zctx).NewScanner(s.ctx, s.filter)
		if err != nil {
			pullersDone()
			rc.Close()
			return nil, err
		}
		pullers = append(pullers, &statScanner{
			scanner:  scanner,
			closer:   rc,
			progress: &s.progress,
		})
	}
	if len(pullers) == 1 {
		return pullers[0], nil
	}
	return merge.New(s.ctx, pullers, importComparator(s.zctx, s.pool).Compare), nil
}

type statScanner struct {
	scanner  zbuf.Scanner
	closer   io.Closer
	err      error
	progress *zbuf.Progress
}

func (s *statScanner) Pull(done bool) (zbuf.Batch, error) {
	if s.scanner == nil {
		return nil, s.err
	}
	batch, err := s.scanner.Pull(done)
	if batch == nil || err != nil {
		s.progress.Add(s.scanner.Progress())
		if err2 := s.closer.Close(); err == nil {
			err = err2
		}
		s.err = err
		s.scanner = nil
	}
	return batch, err
}

type seekFinder func(context.Context, *data.Object) (seekindex.Range, error)

func newSeekFinder(pool *Pool, snap commits.View, f zbuf.Filter) (seekFinder, error) {
	cropped, err := f.AsKeyCroppedByFilter(pool.Layout.Primary(), pool.Layout.Order)
	if err != nil {
		return nil, err
	}
	idx := index.NewFilter(pool.engine, pool.IndexPath, f)
	kf := f.KeyFilter(pool.Layout.Primary())
	cmp := expr.NewValueCompareFn(pool.Layout.Order == order.Asc)
	return func(ctx context.Context, o *data.Object) (seekindex.Range, error) {
		rg := seekindex.Range{End: o.Size}
		var indexSpan extent.Span
		if idx != nil {
			rules, err := snap.LookupIndexObjectRules(o.ID)
			if err != nil && !errors.Is(err, commits.ErrNotFound) {
				return rg, err
			}
			if rules != nil {
				indexSpan, err = idx.Apply(ctx, o.ID, rules)
				if err != nil {
					return rg, err
				}
				if indexSpan == nil {
					rg.End = 0
					return rg, nil
				}
			}
		}
		span := extent.NewGeneric(o.First, o.Last, cmp)
		if indexSpan != nil || cropped != nil && cropped.Eval(span.First(), span.Last()) {
			// If there is an index optimization or the object's span is
			// cropped by the filter then check the seek to find the offset
			// range to scan. Otherwise the entire object can be scanned.
			sr, err := pool.engine.Get(ctx, o.SeekObjectPath(pool.DataPath))
			if err != nil {
				return rg, err
			}
			defer sr.Close()
			r := zngio.NewReader(sr, zed.NewContext())
			return seekindex.Lookup(ctx, r, kf, indexSpan, &o.Last, o.Count, o.Size, pool.Layout.Order)
		}
		return rg, nil
	}, nil
}
