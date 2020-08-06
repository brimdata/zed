package scanner

import (
	"context"
	"fmt"
	"sync"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

// NewMultiScanner returns a Scanner for the logical concatenation of the
// provided readers that filters records by filterExpr and span. Readers are
// read sequentially, and when all records are consumed, Pull will return a nil
// batch and nil error. If any reader returns a non-nil error, Pull will return
// that error.
func NewMultiScanner(ctx context.Context, readers []zbuf.Reader, filterExpr ast.BooleanExpr, span nano.Span) (Scanner, error) {
	var f filter.Filter
	if filterExpr != nil {
		var err error
		if f, err = filter.Compile(filterExpr); err != nil {
			return nil, err
		}
	}
	var scanners []Scanner
	for _, r := range readers {
		var sa ScannerAble
		if f, ok := r.(*zbuf.File); ok {
			sa, _ = f.Reader.(ScannerAble)
		} else {
			sa, _ = r.(ScannerAble)
		}
		if sa != nil {
			s, err := sa.NewScanner(ctx, filterExpr, span)
			if err != nil {
				return nil, err
			}
			scanners = append(scanners, s)
		} else {
			scanners = append(scanners, NewScanner(ctx, r, f, span))
		}
	}
	return &multiScanner{readers: readers, scanners: scanners}, nil
}

type multiScanner struct {
	mu       sync.Mutex
	readers  []zbuf.Reader
	scanners []Scanner
	stats    ScannerStats
}

func (m *multiScanner) Pull() (zbuf.Batch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.scanners) > 0 {
		batch, err := m.scanners[0].Pull()
		if err != nil {
			if _, ok := m.readers[0].(fmt.Stringer); ok {
				err = fmt.Errorf("%s: %w", m.readers[0], err)
			}
			return batch, err
		}
		if batch != nil {
			return batch, nil
		}
		m.stats.Accumulate(m.scanners[0].Stats())
		m.readers = m.readers[1:]
		m.scanners = m.scanners[1:]
	}
	return nil, nil
}

func (m *multiScanner) Stats() *ScannerStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.scanners) == 0 {
		return &m.stats
	}
	s := m.stats
	s.Accumulate(m.scanners[0].Stats())
	return &s
}
