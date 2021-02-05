package lake

import (
	"context"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

// staticSource is an implementation of driver.MultiSource that provides
// a single SpanInfo (with Chunks) to be processed by a zqd worker.
// staticSource is used only for the zqd /worker call.
type staticSource struct {
	*spanMultiSource
	src driver.Source
}

func NewStaticSource(lk *Lake, src driver.Source) driver.MultiSource {
	return &staticSource{
		spanMultiSource: &spanMultiSource{lk: lk},
		src:             src,
	}
}

func (s *staticSource) OrderInfo() (field.Static, bool) {
	return field.New("ts"), s.lk.DataOrder == zbuf.OrderDesc
}

func (s *staticSource) SendSources(ctx context.Context, span nano.Span, srcChan chan driver.Source) error {
	select {
	case srcChan <- s.src:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
