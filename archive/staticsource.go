package archive

import (
	"context"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
)

type staticSource struct {
	ark   *Archive
	chunk Chunk
}

func NewStaticSource(ark *Archive, c Chunk) driver.MultiSource {
	return &staticSource{
		ark:   ark,
		chunk: c,
	}
}

func (s *staticSource) OrderInfo() (string, bool) {
	return "ts", s.ark.DataSortDirection == zbuf.DirTimeReverse
}

func (m *staticSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	so := func() (driver.ScannerCloser, error) {
		si := spanInfo{
			span:   m.chunk.Span(), // make a span from the chunk
			chunks: []Chunk{m.chunk},
		}
		return newSpanScanner(ctx, m.ark, zctx, sf.Filter, sf.FilterExpr, si)
	}
	select {
	case srcChan <- so:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
