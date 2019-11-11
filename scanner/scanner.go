package scanner

import (
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/proc"
)

type Scanner struct {
	reader zson.Reader
}

func NewScanner(reader zson.Reader) *Scanner {
	return &Scanner{
		reader: reader,
	}
}

func (s *Scanner) Pull() (zson.Batch, error) {
	minTs, maxTs := nano.MaxTs, nano.MinTs
	var arr []*zson.Record
	for i := 0; i < 24; i++ {
		tup, err := s.reader.Read()
		if err != nil {
			return nil, err
		}
		if tup == nil {
			break
		}
		if tup.Ts < minTs {
			minTs = tup.Ts
		}
		if tup.Ts > maxTs {
			maxTs = tup.Ts
		}
		// Use tuple.Keep() to copy underlying buffer because call to next
		// reader.Next() will overwrite said buffer.
		arr = append(arr, tup.Keep())
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
