package lake

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"go.uber.org/multierr"
)

type multiCloser []io.Closer

func (c multiCloser) Close() (err error) {
	for _, closer := range c {
		if closeErr := closer.Close(); closeErr != nil {
			err = multierr.Append(err, closeErr)
		}
	}
	return
}

type sortedPuller struct {
	zbuf.Puller
	io.Closer
}

type statScanner struct {
	zbuf.Scanner
	puller zbuf.Puller
	sched  *Scheduler
	err    error
}

func (s *statScanner) Pull() (zbuf.Batch, error) {
	if s.puller == nil {
		return nil, s.err
	}
	batch, err := s.puller.Pull()
	if batch == nil || err != nil {
		s.sched.AddStats(s.Scanner.Stats())
		s.puller = nil
		s.err = err
	}
	return batch, err
}

func newSortedScanner(ctx context.Context, pool *Pool, zctx *zed.Context, filter zbuf.Filter, scan Partition, sched *Scheduler) (*sortedPuller, error) {
	closers := make(multiCloser, 0, len(scan.Objects))
	pullers := make([]zbuf.Puller, 0, len(scan.Objects))
	for _, object := range scan.Objects {
		rc, err := object.NewReader(ctx, pool.engine, pool.DataPath, scan.Span, scan.compare)
		if err != nil {
			closers.Close()
			return nil, err
		}
		closers = append(closers, rc)
		reader := zngio.NewReader(rc, zctx)
		f := filter
		if len(pool.Layout.Keys) != 0 {
			// If the scan span does not wholly contain the data object, then
			// we must filter out records that fall outside the range.
			f = wrapRangeFilter(f, scan.Span, scan.compare, object.First, object.Last, pool.Layout)
		}
		scanner, err := reader.NewScanner(ctx, f)
		if err != nil {
			closers.Close()
			return nil, err
		}
		pullers = append(pullers, &statScanner{
			Scanner: scanner,
			puller:  scanner,
			sched:   sched,
		})
	}
	var merger zbuf.Puller
	if len(pullers) == 1 {
		merger = pullers[0]
	} else {
		merger = zbuf.NewMerger(ctx, pullers, importCompareFn(pool))
	}
	return &sortedPuller{
		Puller: merger,
		Closer: closers,
	}, nil
}

type rangeWrapper struct {
	zbuf.Filter
	first  *zed.Value
	last   *zed.Value
	layout order.Layout
}

func (r *rangeWrapper) AsFilter() (expr.Filter, error) {
	f, err := r.Filter.AsFilter()
	if err != nil {
		return nil, err
	}
	compare := extent.CompareFunc(r.layout.Order)
	return func(ctx expr.Context, rec *zed.Value) bool {
		keyVal, err := rec.Deref(r.layout.Keys[0])
		if err != nil {
			// XXX match keyless records.
			// See issue #2637.
			return true
		}
		if compare(&keyVal, r.first) < 0 || compare(&keyVal, r.last) > 0 {
			return false
		}
		return f == nil || f(ctx, rec)
	}, nil
}

func wrapRangeFilter(f zbuf.Filter, scan extent.Span, cmp expr.ValueCompareFn, first, last zed.Value, layout order.Layout) zbuf.Filter {
	scanFirst := scan.First()
	scanLast := scan.Last()
	if cmp(scanFirst, &first) <= 0 {
		if cmp(scanLast, &last) >= 0 {
			return f
		}
	}
	return &rangeWrapper{
		Filter: f,
		first:  scanFirst,
		last:   scanLast,
		layout: layout,
	}
}
