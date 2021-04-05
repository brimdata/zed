package lake

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

type MultiSource interface {
	driver.MultiSource
	Stats() ScanStats
}

type multiCloser []io.Closer

func (c multiCloser) Close() (err error) {
	for _, closer := range c {
		if closeErr := closer.Close(); closeErr != nil {
			err = multierr.Append(err, closeErr)
		}
	}
	return
}

type pullerCloser struct {
	zbuf.Puller
	zbuf.MultiStats
	io.Closer
}

func newRangeScanner(ctx context.Context, pool *Pool, zctx *zson.Context, sf driver.SourceFilter, scan segment.Partition) (sc *pullerCloser, stats ScanStats, err error) {
	closers := make(multiCloser, 0, len(scan.Segments))
	pullers := make([]zbuf.Puller, 0, len(scan.Segments))
	scanners := make([]zbuf.Scanner, 0, len(scan.Segments))
	span := scan.Span()
	for _, segref := range scan.Segments {
		rc, err := segref.NewReader(ctx, pool.DataPath, span)
		if err != nil {
			closers.Close()
			return nil, stats, err
		}
		stats.TotalBytes += rc.TotalBytes
		stats.ReadBytes += rc.ReadBytes
		closers = append(closers, rc)
		reader := zngio.NewReader(rc, zctx)
		scanner, err := reader.NewScanner(ctx, sf.Filter, span)
		if err != nil {
			closers.Close()
			return nil, stats, err
		}
		scanners = append(scanners, scanner)
		pullers = append(pullers, scanner)
	}
	return &pullerCloser{
		Puller:     zbuf.MergeByTs(ctx, pullers, pool.Order),
		MultiStats: scanners,
		Closer:     closers,
	}, stats, nil
}

// NewMultiSource returns a driver.MultiSource for a Lake. If no alternative
// paths are specified, the MultiSource will send a source for each span in the
// driver.SourceFilter span, and report the same ordering as the archive.
//
// Otherwise, the sources come from localizing the given alternative paths to
// each chunk in the archive, recognizing "_" as the chunk file itself, with no
// defined ordering.
func NewMultiSource(pool *Pool) MultiSource {
	return &spanMultiSource{pool, &ScanStats{}}
}

type spanMultiSource struct {
	pool  *Pool
	stats *ScanStats
}

func (m *spanMultiSource) OrderInfo() (field.Static, bool) {
	return field.New("ts"), m.pool.Order == zbuf.OrderDesc
}

func (m *spanMultiSource) SendSources(ctx context.Context, span nano.Span, srcChan chan driver.Source) error {
	// We keep a channel of []SpanInfos filled to reduce the time
	// query workers are waiting for the next driver.Source.
	const scanPreFetch = 10
	paritionCh := make(chan segment.Partition, scanPreFetch)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		head, err := m.pool.log.Head(ctx)
		if err != nil {
			close(paritionCh)
			return err
		}
		err = head.ScanPartitions(ctx, paritionCh, span)
		close(paritionCh)
		return err
	})
	g.Go(func() error {
		for p := range paritionCh {
			select {
			case srcChan <- &rangeSource{pool: m.pool, partition: p, stats: m.stats}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	return g.Wait()
}

func (m *spanMultiSource) SourceFromRequest(ctx context.Context, req *api.WorkerChunkRequest) (driver.Source, error) {
	return nil, errors.New("TBD: refactor multi-source and how worker requests are created")
}

func (m *spanMultiSource) Stats() ScanStats {
	return m.stats.Copy()
}

type rangeSource struct {
	pool      *Pool
	partition segment.Partition
	stats     *ScanStats
}

func (s *rangeSource) Open(ctx context.Context, zctx *zson.Context, sf driver.SourceFilter) (driver.ScannerCloser, error) {
	scn, stats, err := newRangeScanner(ctx, s.pool, zctx, sf, s.partition)
	s.stats.Accumulate(stats)
	return scn, err
}

func (s *rangeSource) ToRequest(req *api.WorkerChunkRequest) error {
	return errors.New("issue #XXX")
}

type ScanStats struct {
	// TotalBytes is the cumulative size of all the segments accessed.
	TotalBytes int64
	// ReadBytes is the nunmer of bytes read from storage across
	// all segments.  If seek indicies are used this number is generally
	// less than TotalBytes.
	ReadBytes int64
}

func (s *ScanStats) Accumulate(a ScanStats) {
	atomic.AddInt64(&s.TotalBytes, a.TotalBytes)
	atomic.AddInt64(&s.ReadBytes, a.ReadBytes)
}

func (s *ScanStats) Copy() ScanStats {
	return ScanStats{
		TotalBytes: atomic.LoadInt64(&s.TotalBytes),
		ReadBytes:  atomic.LoadInt64(&s.ReadBytes),
	}
}
