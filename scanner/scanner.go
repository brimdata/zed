package scanner

import (
	"context"
	"sync/atomic"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// ScannerAble is implemented by zbuf.Readers that provide an optimized
// implementation of the Scanner interface.
type ScannerAble interface {
	NewScanner(ctx context.Context, filterExpr ast.BooleanExpr, s nano.Span) (Scanner, error)
}

// A Scanner is a zbuf.Batch source that also provides statistics.
type Scanner interface {
	Pull() (zbuf.Batch, error)
	Stats() *ScannerStats
}

// ScannerStats holds Scanner statistics. It should be identical to
// api.ScannerStats.
type ScannerStats struct {
	BytesRead      int64
	BytesMatched   int64
	RecordsRead    int64
	RecordsMatched int64
}

// Accumulate updates its receiver by adding to it the values in ss.
func (s *ScannerStats) Accumulate(ss *ScannerStats) {
	s.BytesRead += ss.BytesRead
	s.BytesMatched += ss.BytesMatched
	s.RecordsRead += ss.RecordsRead
	s.RecordsMatched += ss.RecordsMatched
}

// NewScanner returns a Scanner for reader that filters records by f and s.
func NewScanner(ctx context.Context, reader zbuf.Reader, f filter.Filter, s nano.Span) Scanner {
	return &scanner{
		reader: reader,
		filter: f,
		span:   s,
		ctx:    ctx,
	}
}

type scanner struct {
	reader zbuf.Reader
	filter filter.Filter
	span   nano.Span
	ctx    context.Context
	stats  struct {
		bytesRead      int64
		bytesMatched   int64
		recordsRead    int64
		recordsMatched int64
	}
}

var BatchSize = 100

func (s *scanner) Pull() (zbuf.Batch, error) {
	return zbuf.ReadBatch(s, BatchSize)
}

func (s *scanner) Stats() *ScannerStats {
	return &ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.stats.bytesRead),
		BytesMatched:   atomic.LoadInt64(&s.stats.bytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.stats.recordsRead),
		RecordsMatched: atomic.LoadInt64(&s.stats.recordsMatched),
	}
}

// Read implements zbuf.Reader.Read.
func (s *scanner) Read() (*zng.Record, error) {
	for {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		rec, err := s.reader.Read()
		if err != nil || rec == nil {
			return nil, err
		}
		atomic.AddInt64(&s.stats.bytesRead, int64(len(rec.Raw)))
		atomic.AddInt64(&s.stats.recordsRead, 1)
		if s.span != nano.MaxSpan && !s.span.Contains(rec.Ts()) ||
			s.filter != nil && !s.filter(rec) {
			continue
		}
		atomic.AddInt64(&s.stats.bytesMatched, int64(len(rec.Raw)))
		atomic.AddInt64(&s.stats.recordsMatched, 1)
		// Copy the underlying buffer (if volatile) because next call to
		// reader.Next() may overwrite said buffer.
		rec.CopyBody()
		return rec, nil
	}
}
