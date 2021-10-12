package zbuf

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zio"
)

type Filter interface {
	AsFilter() (expr.Filter, error)
	AsBufferFilter() (*expr.BufferFilter, error)
}

// ScannerAble is implemented by Readers that provide an optimized
// implementation of the Scanner interface.
type ScannerAble interface {
	NewScanner(ctx context.Context, filterExpr Filter) (Scanner, error)
}

// A Statser produces scanner statistics.
type Statser interface {
	Stats() ScannerStats
}

// A Scanner is a Batch source that also provides statistics.
type Scanner interface {
	Statser
	Puller
}

type ScannerCloser interface {
	Scanner
	io.Closer
}

// ScannerStats holds Scanner statistics.
type ScannerStats struct {
	BytesRead      int64 `zed:"bytes_read" json:"bytes_read"`
	BytesMatched   int64 `zed:"bytes_matched" json:"bytes_matched"`
	RecordsRead    int64 `zed:"records_read" json:"records_read"`
	RecordsMatched int64 `zed:"records_matched" json:"records_matched"`
}

// Add updates its receiver by adding to it the values in ss.
func (s *ScannerStats) Add(in ScannerStats) {
	atomic.AddInt64(&s.BytesRead, in.BytesRead)
	atomic.AddInt64(&s.BytesMatched, in.BytesMatched)
	atomic.AddInt64(&s.RecordsRead, in.RecordsRead)
	atomic.AddInt64(&s.RecordsMatched, in.RecordsMatched)
}

func (s *ScannerStats) Copy() ScannerStats {
	return ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.BytesRead),
		BytesMatched:   atomic.LoadInt64(&s.BytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.RecordsRead),
		RecordsMatched: atomic.LoadInt64(&s.RecordsMatched),
	}
}

var ScannerBatchSize = 100

// NewScanner returns a Scanner for r that filters records by filterExpr and s.
func NewScanner(ctx context.Context, r zio.Reader, filterExpr Filter) (Scanner, error) {
	var sa ScannerAble
	if zf, ok := r.(*File); ok {
		sa, _ = zf.Reader.(ScannerAble)
	} else {
		sa, _ = r.(ScannerAble)
	}
	if sa != nil {
		return sa.NewScanner(ctx, filterExpr)
	}
	var f expr.Filter
	if filterExpr != nil {
		var err error
		if f, err = filterExpr.AsFilter(); err != nil {
			return nil, err
		}
	}
	sc := &scanner{reader: r, filter: f, ctx: ctx}
	sc.Puller = NewPuller(sc, ScannerBatchSize)
	return sc, nil
}

type scanner struct {
	Puller
	reader zio.Reader
	filter expr.Filter
	ctx    context.Context

	stats ScannerStats
}

func (s *scanner) Stats() ScannerStats {
	return s.stats.Copy()
}

// Read implements Reader.Read.
func (s *scanner) Read() (*zed.Record, error) {
	for {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		rec, err := s.reader.Read()
		if err != nil || rec == nil {
			return nil, err
		}
		atomic.AddInt64(&s.stats.BytesRead, int64(len(rec.Bytes)))
		atomic.AddInt64(&s.stats.RecordsRead, 1)
		if s.filter != nil && !s.filter(rec) {
			continue
		}
		atomic.AddInt64(&s.stats.BytesMatched, int64(len(rec.Bytes)))
		atomic.AddInt64(&s.stats.RecordsMatched, 1)
		// Copy the underlying buffer (if volatile) because next call to
		// reader.Next() may overwrite said buffer.
		rec.CopyBytes()
		return rec, nil
	}
}

type MultiStats []Scanner

func (m MultiStats) Stats() ScannerStats {
	var ss ScannerStats
	for _, s := range m {
		ss.Add(s.Stats())
	}
	return ss
}

func NamedScanner(s Scanner, name string) *namedScanner {
	return &namedScanner{
		Scanner: s,
		name:    name,
	}
}

type namedScanner struct {
	Scanner
	name string
}

func (n *namedScanner) Pull() (Batch, error) {
	b, err := n.Scanner.Pull()
	if err != nil {
		err = fmt.Errorf("%s: %w", n.name, err)
	}
	return b, err
}

func ScannerNopCloser(s Scanner) *nopCloser {
	return &nopCloser{s}
}

type nopCloser struct {
	Scanner
}

func (n *nopCloser) Close() error {
	return nil
}
