package scanner

import (
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Scanner struct {
	reader zbuf.Reader
	filter filter.Filter
	span   nano.Span
}

func NewScanner(reader zbuf.Reader, f filter.Filter) *Scanner {
	return &Scanner{
		reader: reader,
		filter: f,
	}
}

func (s *Scanner) SetSpan(span nano.Span) {
	s.span = span
}

const batchSize = 100

// Pull implements Proc.Pull.
func (s *Scanner) Pull() (zbuf.Batch, error) {
	return zbuf.ReadBatch(s, batchSize)
}

// Read implements zbuf.Reader.Read.
func (s *Scanner) Read() (*zng.Record, error) {
	for {
		rec, err := s.reader.Read()
		if err != nil || rec == nil {
			return nil, err
		}
		if s.span.Dur != 0 && !s.span.Contains(rec.Ts) ||
			s.filter != nil && !s.filter(rec) {
			continue
		}
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
