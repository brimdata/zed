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
	NewScanner(ctx context.Context, f filter.Filter, filterExpr ast.BooleanExpr, s nano.Span) (Scanner, error)
}

// A ScannerStatsAble generates scanner statistics.
type ScannerStatsAble interface {
	Stats() *ScannerStats
}

// A Scanner is a zbuf.Batch source that also provides statistics.
type Scanner interface {
	ScannerStatsAble
	Pull() (zbuf.Batch, error)
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

// NewScanner returns a Scanner for r that filters records by filterExpr and s.
func NewScanner(ctx context.Context, r zbuf.Reader, f filter.Filter, filterExpr ast.BooleanExpr, s nano.Span) (Scanner, error) {
	var sa ScannerAble
	if zf, ok := r.(*zbuf.File); ok {
		sa, _ = zf.Reader.(ScannerAble)
	} else {
		sa, _ = r.(ScannerAble)
	}
	if sa != nil {
		return sa.NewScanner(ctx, f, filterExpr, s)
	}
	return &scanner{reader: r, filter: f, span: s, ctx: ctx}, nil
}

type scanner struct {
	reader zbuf.Reader
	filter filter.Filter
	span   nano.Span
	ctx    context.Context
	stats  ScannerStats
}

var BatchSize = 100

func (s *scanner) Pull() (zbuf.Batch, error) {
	return zbuf.ReadBatch(s, BatchSize)
}

func (s *scanner) Stats() *ScannerStats {
	return &ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.stats.BytesRead),
		BytesMatched:   atomic.LoadInt64(&s.stats.BytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.stats.RecordsRead),
		RecordsMatched: atomic.LoadInt64(&s.stats.RecordsMatched),
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
		atomic.AddInt64(&s.stats.BytesRead, int64(len(rec.Raw)))
		atomic.AddInt64(&s.stats.RecordsRead, 1)
		if s.span != nano.MaxSpan && !s.span.Contains(rec.Ts()) ||
			s.filter != nil && !s.filter(rec) {
			continue
		}
		atomic.AddInt64(&s.stats.BytesMatched, int64(len(rec.Raw)))
		atomic.AddInt64(&s.stats.RecordsMatched, 1)
		// Copy the underlying buffer (if volatile) because next call to
		// reader.Next() may overwrite said buffer.
		rec.CopyBody()
		return rec, nil
	}
}
