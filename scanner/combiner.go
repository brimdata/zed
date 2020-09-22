package scanner

import (
	"context"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type combinerScanner struct {
	combiner zbuf.Reader
	scanners []Scanner
}

// NewCombinerScanner returns a Scanner that combines the records scanned from
// a set of filtered readers.
func NewCombinerScanner(ctx context.Context, readers []zbuf.Reader, spans []nano.Span, cmp zbuf.RecordCmpFn, f filter.Filter, filterExpr ast.BooleanExpr) (Scanner, error) {
	if len(readers) != len(spans) {
		panic("length mismatch between readers and spans")
	}
	scanners := make([]Scanner, len(readers))
	scanReaders := make([]zbuf.Reader, len(readers))
	for i, r := range readers {
		s, err := NewScanner(ctx, r, f, filterExpr, spans[i])
		if err != nil {
			return nil, err
		}
		scanners[i] = s
		scanReaders[i] = zbuf.PullerReader(s)
	}
	return &combinerScanner{
		combiner: zbuf.NewCombiner(scanReaders, cmp),
		scanners: scanners,
	}, nil
}

func (c *combinerScanner) Pull() (zbuf.Batch, error) {
	return zbuf.ReadBatch(c, BatchSize)
}

func (c *combinerScanner) Read() (*zng.Record, error) {
	return c.combiner.Read()
}

func (c *combinerScanner) Stats() *ScannerStats {
	var ss ScannerStats
	for _, s := range c.scanners {
		ss.Accumulate(s.Stats())
	}
	return &ss
}
