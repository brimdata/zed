package lake

import (
	"context"
	"io"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
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
	span := scan.Span()
	for _, segref := range scan.Segments {
		rc, err := segref.NewReader(ctx, pool.engine, pool.DataPath, span)
		if err != nil {
			closers.Close()
			return nil, err
		}
		closers = append(closers, rc)
		reader := zngio.NewReader(rc, zctx)
		scanner, err := reader.NewScanner(ctx, filter, span)
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
