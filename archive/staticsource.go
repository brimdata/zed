package archive

import (
	"context"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

// staticSource is an implementation of driver.MultiSource that provides
// a single SpanInfo (with Chunks) to be processed by a zqd worker.
// staticSource is used only for the zqd /worker call.
type staticSource struct {
	ark *Archive
	sis SpanInfoSource
}

type SpanInfoSource struct {
	Span       nano.Span
	ChunkPaths []string
}

func NewStaticSource(ark *Archive, sis SpanInfoSource) driver.MultiSource {
	return &staticSource{
		ark: ark,
		sis: sis,
	}
}

func (s *staticSource) OrderInfo() (field.Static, bool) {
	return field.New("ts"), s.ark.DataOrder == zbuf.OrderDesc
}

func (s *staticSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	si := SpanInfo{Span: s.sis.Span}
	for _, p := range s.sis.ChunkPaths {
		tsd, _, id, ok := parseChunkRelativePath(p)
		if !ok {
			return zqe.E(zqe.Invalid, "invalid chunk path: %v", p)
		}
		md, err := readChunkMetadata(ctx, chunkMetadataPath(s.ark, tsd, id))
		if err != nil {
			return err
		}
		si.Chunks = append(si.Chunks, md.Chunk(id))
	}
	so := func() (driver.ScannerCloser, error) {
		return newSpanScanner(ctx, s.ark, zctx, sf.Filter, sf.FilterExpr, si)
	}
	select {
	case srcChan <- so:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
