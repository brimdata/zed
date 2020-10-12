package archive

import (
	"context"

	"github.com/brimsec/zq/multisource"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
)

// staticSource is an implemetation of multisource.MultiSource that provides
// a single SpanInfo (with Chunks) to be processed by a zqd worker.
// staticSource is used only for the zqd /worker call.
type staticSource struct {
	ark *Archive
	si  SpanInfo
}

func NewStaticSource(ark *Archive, si SpanInfo) multisource.MultiSource {
	return &staticSource{
		ark: ark,
		si:  si,
	}
}

func (s *staticSource) OrderInfo() (string, bool) {
	return "ts", s.ark.DataSortDirection == zbuf.DirTimeReverse
}

func (s *staticSource) SendSources(ctx context.Context, zctx *resolver.Context, sf multisource.SourceFilter, srcChan chan multisource.Source) error {
	so := func() (multisource.ScannerCloser, error) {
		return newSpanScanner(ctx, s.ark, zctx, sf.Filter, sf.FilterExpr, s.si)
	}
	select {
	case srcChan <- so:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
