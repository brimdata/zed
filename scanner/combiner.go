package scanner

import (
	"context"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

type combiner struct {
	reader   zbuf.Reader
	scanners []Scanner
}

// NewCombiner returns a Scanner that combines the records scanned from
// a set of filtered readers.
func NewCombiner(ctx context.Context, readers []zbuf.Reader, less zbuf.RecordLessFn, filterExpr ast.BooleanExpr, span nano.Span) (Scanner, error) {
	scanners := make([]Scanner, len(readers))
	scanReaders := make([]zbuf.Reader, len(readers))
	for i, r := range readers {
		s, err := NewScanner(ctx, r, filterExpr, span)
		if err != nil {
			return nil, err
		}
		scanners[i] = s
		scanReaders[i] = zbuf.PullerReader(s)
	}
	return &combiner{
		reader:   zbuf.NewCombiner(scanReaders, less),
		scanners: scanners,
	}, nil
}

func (c *combiner) Pull() (zbuf.Batch, error) {
	return zbuf.ReadBatch(c.reader, BatchSize)
}

func (c *combiner) Stats() *ScannerStats {
	var ss ScannerStats
	for _, s := range c.scanners {
		ss.Accumulate(s.Stats())
	}
	return &ss
}
