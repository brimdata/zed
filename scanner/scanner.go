package scanner

import (
	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/proc"
)

type Scanner struct {
	reader zng.Reader
	filter filter.Filter
}

func NewScanner(reader zng.Reader, f filter.Filter) *Scanner {
	return &Scanner{
		reader: reader,
		filter: f,
	}
}

const batchSize = 100

func (s *Scanner) Pull() (zng.Batch, error) {
	minTs, maxTs := nano.MaxTs, nano.MinTs
	var arr []*zng.Record
	match := s.filter
	for len(arr) < batchSize {
		rec, err := s.reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if match != nil && !match(rec) {
			continue
		}
		if rec.Ts < minTs {
			minTs = rec.Ts
		}
		if rec.Ts > maxTs {
			maxTs = rec.Ts
		}
		// Copy the underlying buffer (if volatile) because call to next
		// reader.Next() may overwrite said buffer.
		rec.CopyBody()
		arr = append(arr, rec)
	}
	if arr == nil {
		return nil, nil
	}
	span := nano.NewSpanTs(minTs, maxTs)
	return zng.NewArray(arr, span), nil
}

// Done is required to implement proc.Proc interface. Ignore for now.
func (s *Scanner) Done() {}

// Parents is required to implement proc.Proc interface. Since Scanner will
// always be the head of a flowgraph this should always return nil.
func (s *Scanner) Parents() []proc.Proc { return nil }
