package scanner

import (
	"sync/atomic"

	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zqd/api"
)

// Scanner implements the proc.Proc interface.
type Scanner struct {
	reader zbuf.Reader
	filter filter.Filter
	span   nano.Span
	stats  struct {
		currentTs      int64
		bytesRead      int64
		bytesMatched   int64
		recordsRead    int64
		recordsMatched int64
	}
}

func NewScanner(reader zbuf.Reader, f filter.Filter, s nano.Span) *Scanner {
	return &Scanner{
		reader: reader,
		filter: f,
		span:   s,
	}
}

const batchSize = 100

// Pull implements Proc.Pull.
func (s *Scanner) Pull() (zbuf.Batch, error) {
	return zbuf.ReadBatch(s, batchSize)
}

func (s *Scanner) Stats() api.ScannerStats {
	return api.ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.stats.bytesRead),
		BytesMatched:   atomic.LoadInt64(&s.stats.bytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.stats.recordsRead),
		RecordsMatched: atomic.LoadInt64(&s.stats.recordsMatched),
	}
}

// Read implements zbuf.Reader.Read.
func (s *Scanner) Read() (*zng.Record, error) {
	for {
		rec, err := s.reader.Read()
		if err != nil || rec == nil {
			return nil, err
		}
		atomic.AddInt64(&s.stats.bytesRead, int64(len(rec.Raw)))
		atomic.AddInt64(&s.stats.recordsRead, 1)
		if s.span != nano.MaxSpan && !s.span.Contains(rec.Ts) ||
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

// Done is required to implement proc.Proc interface. Ignore for now.
func (s *Scanner) Done() {}

// Parents is required to implement proc.Proc interface. Since Scanner will
// always be the head of a flowgraph this should always return nil.
func (s *Scanner) Parents() []proc.Proc { return nil }
