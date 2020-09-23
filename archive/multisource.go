package archive

import (
	"context"
	"io"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
)

// A SpanInfo is a logical view of the records within a time span, stored
// in one or more Chunks.
type SpanInfo struct {
	First nano.Ts // timestamp of first record in this span
	Last  nano.Ts // timestamp of last record in this span

	// Chunks are the data files that contain records within this SpanInfo.
	// The Chunks may have spans that extend beyond this SpanInfo, so any
	// records from these Chunks should be limited to those that fall within a
	// closed span constructed from First & Last.
	Chunks []Chunk
}

// Span returns an inclusive nano.Span that contains both the first
// and last record timestamps.
func (si SpanInfo) Span() nano.Span {
	return closedSpan(si.First, si.Last)
}

func spanWalk(ctx context.Context, ark *Archive, filter nano.Span, v func(si SpanInfo) error) error {
	return tsDirVisit(ctx, ark, filter, func(_ tsDir, chunks []Chunk) error {
		sinfos := mergeChunksToSpans(chunks, ark.DataSortDirection, filter)
		for _, s := range sinfos {
			if err := v(s); err != nil {
				return err
			}
		}
		return nil
	})
}

type multiCloser struct {
	closers []io.Closer
}

func (c *multiCloser) Close() error {
	var err error
	for _, c := range c.closers {
		if closeErr := c.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

type scannerCloser struct {
	scanner.Scanner
	io.Closer
}

func newSpanScanner(ctx context.Context, ark *Archive, zctx *resolver.Context, f filter.Filter, filterExpr ast.BooleanExpr, si SpanInfo) (sc *scannerCloser, err error) {
	if len(si.Chunks) == 1 {
		rc, err := iosrc.NewReader(ctx, si.Chunks[0].Path(ark))
		if err != nil {
			return nil, err
		}
		sn, err := scanner.NewScanner(ctx, zngio.NewReader(rc, zctx), f, filterExpr, si.Span())
		if err != nil {
			rc.Close()
			return nil, err
		}
		return &scannerCloser{sn, rc}, nil
	}
	closers := make([]io.Closer, len(si.Chunks))
	defer func() {
		if err != nil {
			for _, c := range closers {
				c.Close()
			}
		}
	}()
	readers := make([]zbuf.Reader, len(si.Chunks))
	for i, chunk := range si.Chunks {
		rc, err := iosrc.NewReader(ctx, chunk.Path(ark))
		if err != nil {
			return nil, err
		}
		closers[i] = rc
		readers[i] = zngio.NewReader(rc, zctx)
	}
	sn, err := scanner.NewCombiner(ctx, readers, zbuf.RecordCompare(ark.DataSortDirection), f, filterExpr, si.Span())
	if err != nil {
		return nil, err
	}
	return &scannerCloser{
		Scanner: sn,
		Closer:  &multiCloser{closers},
	}, nil
}

type multiSource struct {
	ark      *Archive
	altPaths []string
}

// NewMultiSource returns a driver.MultiSource for an Archive. If no alternative
// paths are specified, the MultiSource will send a source for each span in the
// driver.SourceFilter span, and report the same ordering as the archive.
//
// Otherwise, the sources come from localizing the given alternative paths to
// each chunk in the archive, recognizing "_" as the chunk file itself, with no
// defined ordering.
func NewMultiSource(ark *Archive, altPaths []string) driver.MultiSource {
	if len(altPaths) == 1 && altPaths[0] == "_" {
		altPaths = nil
	}
	return &multiSource{
		ark:      ark,
		altPaths: altPaths,
	}
}

func (ams *multiSource) OrderInfo() (string, bool) {
	if len(ams.altPaths) == 0 {
		return "ts", ams.ark.DataSortDirection == zbuf.DirTimeReverse
	}
	return "", false
}

func (ams *multiSource) spanWalk(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	return spanWalk(ctx, ams.ark, sf.Span, func(si SpanInfo) error {
		so := func() (driver.ScannerCloser, error) {
			return newSpanScanner(ctx, ams.ark, zctx, sf.Filter, sf.FilterExpr, si)
		}
		select {
		case srcChan <- so:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (ams *multiSource) chunkWalk(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	return Walk(ctx, ams.ark, func(chunk Chunk) error {
		so := func() (driver.ScannerCloser, error) {
			paths := make([]string, len(ams.altPaths))
			for i, input := range ams.altPaths {
				paths[i] = chunk.Localize(ams.ark, input).String()
			}
			rc := detector.MultiFileReader(zctx, paths, zio.ReaderOpts{Format: "zng"})
			sn, err := scanner.NewScanner(ctx, rc, sf.Filter, sf.FilterExpr, sf.Span)
			if err != nil {
				return nil, err
			}
			return &scannerCloser{Scanner: sn, Closer: rc}, nil
		}
		select {
		case srcChan <- so:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (ams *multiSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	if len(ams.altPaths) == 0 {
		return ams.spanWalk(ctx, zctx, sf, srcChan)
	}
	return ams.chunkWalk(ctx, zctx, sf, srcChan)
}
