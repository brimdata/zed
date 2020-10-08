package archive

import (
	"context"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
)

type staticSource struct {
	ark *Archive
	si  SpanInfo
}

func NewStaticSource(ark *Archive, si SpanInfo) driver.MultiSource {
	return &staticSource{
		ark: ark,
		si:  si,
	}
}

func (s *staticSource) OrderInfo() (string, bool) {
	return "ts", s.ark.DataSortDirection == zbuf.DirTimeReverse
}

func (s *staticSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	so := func() (driver.ScannerCloser, error) {
		return newSpanScanner(ctx, s.ark, zctx, sf.Filter, sf.FilterExpr, s.si)
	}
	// suggestion: don't send a closure here, send a SpanInfo
	select {
	case srcChan <- so:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
