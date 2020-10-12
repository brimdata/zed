package archive

import (
	"context"
	"errors"
	"io"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
	"github.com/segmentio/ksuid"
)

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
	if len(altPaths) != 0 {
		return &chunkMultiSource{
			ark:      ark,
			altPaths: altPaths,
		}
	}
	return &spanMultiSource{
		ark: ark,
	}
}

type spanMultiSource struct {
	ark *Archive
}

func (m *spanMultiSource) OrderInfo() (field.Static, bool) {
	return field.New("ts"), m.ark.DataSortDirection == zbuf.DirTimeReverse
}

func (m *spanMultiSource) SendSources(ctx context.Context, span nano.Span, srcChan chan driver.Source) error {
	return SpanWalk(ctx, m.ark, span, func(si SpanInfo) error {
		select {
		case srcChan <- &spanSource{ark: m.ark, spanInfo: si}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (m *spanMultiSource) SourceFromRequest(req *api.WorkerRequest) (driver.Source, error) {
	chunks := make([]Chunk, len(req.Chunks))
	for i, chunk := range req.Chunks {
		id, err := ksuid.Parse(chunk.Id)
		if err != nil {
			return nil, zqe.E(zqe.Invalid, "unparsable ksuid")
		}
		chunks[i].Id = id
		chunks[i].First = chunk.First
		chunks[i].Last = chunk.Last
		chunks[i].Kind = FileKind(chunk.Kind)
		chunks[i].RecordCount = chunk.RecordCount
	}
	return &spanSource{
		ark: m.ark,
		spanInfo: SpanInfo{
			Span:   req.Span,
			Chunks: chunks,
		},
	}, nil
}

type spanSource struct {
	ark      *Archive
	spanInfo SpanInfo
}

func (s *spanSource) Open(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter) (driver.ScannerCloser, error) {
	return newSpanScanner(ctx, s.ark, zctx, sf.Filter, sf.FilterExpr, s.spanInfo)
}

func (s *spanSource) ToRequest(req *api.WorkerRequest) error {
	req.Span = s.spanInfo.Span
	for _, c := range s.spanInfo.Chunks {
		req.Chunks = append(req.Chunks, api.Chunk{
			Id:          c.Id.String(),
			First:       c.First,
			Last:        c.Last,
			Kind:        string(c.Kind),
			RecordCount: c.RecordCount,
		})
	}
	return nil
}

// A chunkMultiSource uses the archive.Walk call to provide a driver.Source
// for each chunk in the archive, possibly combining its data with files named
// by altPaths located in the chunk's zar directory.
type chunkMultiSource struct {
	ark      *Archive
	altPaths []string
}

func (cms *chunkMultiSource) OrderInfo() (field.Static, bool) {
	return nil, false
}

func (cms *chunkMultiSource) SendSources(ctx context.Context, span nano.Span, srcChan chan driver.Source) error {
	return Walk(ctx, cms.ark, func(chunk Chunk) error {
		select {
		case srcChan <- &chunkSource{cms: cms, chunk: chunk}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

var errReqForChunk = errors.New("no request support for chunk sources")

func (cms *chunkMultiSource) SourceFromRequest(req *api.WorkerRequest) (driver.Source, error) {
	return nil, errReqForChunk
}

type chunkSource struct {
	cms   *chunkMultiSource
	chunk Chunk
}

func (s *chunkSource) Open(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter) (driver.ScannerCloser, error) {
	paths := make([]string, len(s.cms.altPaths))
	for i, input := range s.cms.altPaths {
		paths[i] = s.chunk.Localize(s.cms.ark, input).String()
	}
	rc := detector.MultiFileReader(zctx, paths, zio.ReaderOpts{Format: "zng"})
	sn, err := scanner.NewScanner(ctx, rc, sf.Filter, sf.FilterExpr, sf.Span)
	if err != nil {
		return nil, err
	}
	return &scannerCloser{Scanner: sn, Closer: rc}, nil
}

func (s *chunkSource) ToRequest(*api.WorkerRequest) error {
	return errReqForChunk
}
