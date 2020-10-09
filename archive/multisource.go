package archive

import (
	"context"
	"io"

	"github.com/brimsec/zq/ast"
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
	Span nano.Span

	// chunks are the data files that contain records within this SpanInfo.
	// The Chunks may have spans that extend beyond this SpanInfo, so any
	// records from these Chunks should be limited to those that fall within a
	// closed span constructed from First & Last.
	Chunks []Chunk
}

func (s SpanInfo) GetSpan() nano.Span { return s.Span }
func (s SpanInfo) GetChunks() []address.Chunk {
	// It would be nice to know a prettier way to do this
	var chunks = make([]address.Chunk, len(s.Chunks))
	for i := 0; i < len(s.Chunks); i++ {
		chunks[i] = s.Chunks[i]
	}
	return chunks
}

// compile time check
var _ driver.SpanInfo = SpanInfo{}

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

func NewSpanScanner(ctx context.Context, ark *Archive, zctx *resolver.Context, f filter.Filter, filterExpr ast.BooleanExpr, si SpanInfo) (sc *scannerCloser, err error) {
	if len(si.Chunks) == 1 {
		rc, err := iosrc.NewReader(ctx, si.Chunks[0].Path(ark))
		if err != nil {
			return nil, err
		}
		sn, err := scanner.NewScanner(ctx, zngio.NewReader(rc, zctx), f, filterExpr, si.Span)
		if err != nil {
			rc.Close()
			return nil, err
		}
		return &scannerCloser{sn, rc}, nil
	}
	closers := make([]io.Closer, 0, len(si.Chunks))
	defer func() {
		if err != nil {
			for _, c := range closers {
				c.Close()
			}
		}
	}()
	readers := make([]zbuf.Reader, 0, len(si.Chunks))
	for _, chunk := range si.Chunks {
		rc, err := iosrc.NewReader(ctx, chunk.Path(ark))
		if err != nil {
			return nil, err
		}
		closers = append(closers, rc)
		readers = append(readers, zngio.NewReader(rc, zctx))
	}
	sn, err := scanner.NewCombiner(ctx, readers, zbuf.RecordCompare(ark.DataSortDirection), f, filterExpr, si.Span)
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

func (m *multiSource) GetAltPaths() []string { return altPaths }

// compile time check
var _ driver.MultiSource = multiSource{}

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

func (m *multiSource) OrderInfo() (string, bool) {
	if len(m.altPaths) == 0 {
		return "ts", m.ark.DataSortDirection == zbuf.DirTimeReverse
	}
	return "", false
}

func (m *multiSource) spanWalk(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan<- driver.SourceOpener) error {
	return spanWalk(ctx, m.ark, sf.Span, func(si SpanInfo) error {
		so := func() (driver.ScannerCloser, error) {
			println("archive.multisource spanWalk function called ", sf.FilterExpr)
			return newSpanScanner(ctx, m.ark, zctx, sf.Filter, sf.FilterExpr, si)
		}
		// suggest: put the SpanInfo on the channel here
		select {
		case srcChan <- so:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (m *multiSource) chunkWalk(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan<- driver.SourceOpener) error {
	return Walk(ctx, m.ark, func(chunk Chunk) error {
		so := func() (driver.ScannerCloser, error) {
			paths := make([]string, len(m.altPaths))
			for i, input := range m.altPaths {
				paths[i] = chunk.Localize(m.ark, input).String()
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

func (m *multiSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	if len(m.altPaths) == 0 {
		return m.spanWalk(ctx, zctx, sf, srcChan)
	}
	return m.chunkWalk(ctx, zctx, sf, srcChan)
}
