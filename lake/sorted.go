package lake

import (
	"context"
	"io"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
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

func newSortedScanner(ctx context.Context, pool *Pool, zctx *zson.Context, filter zbuf.Filter, scan Partition, sched *Scheduler) (*sortedPuller, error) {
	closers := make(multiCloser, 0, len(scan.Segments))
	pullers := make([]zbuf.Puller, 0, len(scan.Segments))
	for _, segref := range scan.Segments {
		rc, err := segref.NewReader(ctx, pool.engine, pool.DataPath, scan.Span, scan.compare)
		if err != nil {
			closers.Close()
			return nil, err
		}
		closers = append(closers, rc)
		reader := zngio.NewReader(rc, zctx)
		f := filter
		if len(pool.Layout.Keys) != 0 {
			// If the scan span does not wholly contain the segment, then
			// we must filter out records that fall outside the range.
			f = wrapRangeFilter(f, scan.Span, scan.compare, segref.First, segref.Last, pool.Layout.Keys[0])
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
	return &sortedPuller{
		Puller: zbuf.MergeByTs(ctx, pullers, pool.Layout.Order),
		Closer: closers,
	}, nil
}

type rangeWrapper struct {
	zbuf.Filter
	first   zng.Value
	last    zng.Value
	key     field.Path
	compare expr.ValueCompareFn
}

func (r *rangeWrapper) AsFilter() (expr.Filter, error) {
	f, err := r.Filter.AsFilter()
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) bool {
		keyVal, err := rec.Deref(r.key)
		if err != nil {
			// XXX match keyless records.
			// See issue #2637.
			return true
		}
		if r.compare(keyVal, r.first) < 0 || r.compare(keyVal, r.last) > 0 {
			return false
		}
		return f == nil || f(rec)
	}, nil
}

func wrapRangeFilter(f zbuf.Filter, scan extent.Span, cmp expr.ValueCompareFn, first, last zng.Value, key field.Path) zbuf.Filter {
	scanFirst := scan.First()
	scanLast := scan.Last()
	if cmp(scanFirst, first) <= 0 {
		if cmp(scanLast, last) >= 0 {
			return f
		}
	}
	return &rangeWrapper{
		Filter:  f,
		first:   scanFirst,
		last:    scanLast,
		key:     key,
		compare: cmp,
	}
}
