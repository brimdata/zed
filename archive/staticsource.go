package archive

import (
	"context"

	"github.com/brimsec/zq/address"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
)

// staticSource is an implemetation of driver.MultiSource that provides
// a single SpanInfo (with Chunks) to be processed by a zqd worker.
// staticSource is used only for the zqd /worker call.
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

func (s *staticSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan address.SpanInfo) error {
	s.si.opener = func() (scanner.ScannerCloser, error) {
		return newSpanScanner(ctx, s.ark, zctx, sf.Filter, sf.FilterExpr, s.si)
	}
	select {
	case srcChan <- s.si:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
