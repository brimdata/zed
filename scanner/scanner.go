package scanner

import (
	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/proc"
)

type Scanner struct {
	reader zson.Reader
	filter filter.Filter
}

func NewScanner(reader zson.Reader, f filter.Filter) *Scanner {
	return &Scanner{
		reader: reader,
		filter: f,
	}
}

const batchSize = 24

func (s *Scanner) Pull() (zson.Batch, error) {
	minTs, maxTs := nano.MaxTs, nano.MinTs
	var arr []*zson.Record
	match := s.filter
	for len(arr) < batchSize {
		rec, err := s.reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if rec.IsControl() || (match != nil && !match(rec)) {
			continue
		}
		if rec.Ts < minTs {
			minTs = rec.Ts
		}
		if rec.Ts > maxTs {
			maxTs = rec.Ts
		}
		// Use rec.Keep() to copy underlying buffer because call to next
		// reader.Next() may overwrite said buffer.
		arr = append(arr, rec.Keep())
	}
	if arr == nil {
		return nil, nil
	}
	span := nano.NewSpanTs(minTs, maxTs)
	return zson.NewArray(arr, span), nil
}

// Done is required to implement proc.Proc interface. Ignore for now.
func (s *Scanner) Done() {}

// Parents is required to implement proc.Proc interface. Since Scanner will
// always be the head of a flowgraph this should always return nil.
func (s *Scanner) Parents() []proc.Proc { return nil }
