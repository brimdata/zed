package archive

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/multierr"
)

type MultiSource interface {
	driver.MultiSource
	Stats() ChunkStats
}

type multiCloser []io.Closer

func (c multiCloser) Close() (err error) {
	for _, closer := range c {
		if closeErr := closer.Close(); closeErr != nil {
			err = multierr.Append(err, closeErr)
		}
	}
	return
}

type scannerCloser struct {
	scanner.Scanner
	io.Closer
}

func newSpanScanner(ctx context.Context, ark *Archive, zctx *resolver.Context, sf driver.SourceFilter, si SpanInfo) (sc *scannerCloser, stats ChunkStats, err error) {
	closers := make(multiCloser, 0, len(si.Chunks))
	readers := make([]zbuf.Reader, 0, len(si.Chunks))
	for _, chunk := range si.Chunks {
		rc, err := newChunkReader(ctx, chunk, ark, si.Span)
		if err != nil {
			closers.Close()
			return nil, stats, err
		}
		stats.ChunksOpenedBytes += rc.totalSize
		stats.ChunksReadBytes += rc.readSize
		closers = append(closers, rc)
		readers = append(readers, zngio.NewReader(rc, zctx))
	}
	var scn scanner.Scanner
	if len(readers) == 1 {
		scn, err = scanner.NewScanner(ctx, readers[0], sf.Filter, sf.FilterExpr, si.Span)
	} else {
		scn, err = scanner.NewCombiner(ctx, readers, zbuf.RecordCompare(ark.DataOrder), sf.Filter, sf.FilterExpr, si.Span)
	}
	if err != nil {
		closers.Close()
		return nil, stats, err
	}
	return &scannerCloser{
		Scanner: scn,
		Closer:  closers,
	}, stats, nil
}

// NewMultiSource returns a driver.MultiSource for an Archive. If no alternative
// paths are specified, the MultiSource will send a source for each span in the
// driver.SourceFilter span, and report the same ordering as the archive.
//
// Otherwise, the sources come from localizing the given alternative paths to
// each chunk in the archive, recognizing "_" as the chunk file itself, with no
// defined ordering.
func NewMultiSource(ark *Archive, altPaths []string) MultiSource {
	if len(altPaths) == 1 && altPaths[0] == "_" {
		altPaths = nil
	}
	if len(altPaths) != 0 {
		return &chunkMultiSource{
			ark:      ark,
			altPaths: altPaths,
			stats:    &ChunkStats{},
		}
	}
	return &spanMultiSource{ark, &ChunkStats{}}
}

type spanMultiSource struct {
	ark   *Archive
	stats *ChunkStats
}

func (m *spanMultiSource) OrderInfo() (field.Static, bool) {
	return field.New("ts"), m.ark.DataOrder == zbuf.OrderDesc
}

func (m *spanMultiSource) SendSources(ctx context.Context, span nano.Span, srcChan chan driver.Source) error {
	return SpanWalk(ctx, m.ark, span, func(si SpanInfo) error {
		select {
		case srcChan <- &spanSource{ark: m.ark, spanInfo: si, stats: m.stats}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (m *spanMultiSource) SourceFromRequest(ctx context.Context, req *api.WorkerRequest) (driver.Source, error) {
	si := SpanInfo{Span: req.Span}
	for _, p := range req.ChunkPaths {
		tsd, _, id, ok := parseChunkRelativePath(p)
		if !ok {
			return nil, zqe.E(zqe.Invalid, "invalid chunk path: %v", p)
		}
		md, err := readChunkMetadata(ctx, chunkMetadataPath(m.ark, tsd, id))
		if err != nil {
			return nil, err
		}
		si.Chunks = append(si.Chunks, md.Chunk(id))
	}
	return &spanSource{
		ark:      m.ark,
		spanInfo: si,
		stats:    m.stats,
	}, nil
}

func (m *spanMultiSource) Stats() ChunkStats {
	return m.stats.Copy()
}

type spanSource struct {
	ark      *Archive
	spanInfo SpanInfo
	stats    *ChunkStats
}

func (s *spanSource) Open(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter) (driver.ScannerCloser, error) {
	scn, stats, err := newSpanScanner(ctx, s.ark, zctx, sf, s.spanInfo)
	s.stats.Accumulate(stats)
	return scn, err
}

func (s *spanSource) ToRequest(req *api.WorkerRequest) error {
	req.Span = s.spanInfo.Span
	req.DataPath = s.ark.DataPath.String()
	for _, c := range s.spanInfo.Chunks {
		req.ChunkPaths = append(req.ChunkPaths, c.RelativePath())
	}
	return nil
}

// A chunkMultiSource uses the archive.Walk call to provide a driver.Source
// for each chunk in the archive, possibly combining its data with files named
// by altPaths located in the chunk's zar directory.
type chunkMultiSource struct {
	ark      *Archive
	altPaths []string
	stats    *ChunkStats
}

func (cms *chunkMultiSource) OrderInfo() (field.Static, bool) {
	return nil, false
}

func (cms *chunkMultiSource) SendSources(ctx context.Context, span nano.Span, srcChan chan driver.Source) error {
	return Walk(ctx, cms.ark, func(chunk Chunk) error {
		select {
		case srcChan <- &chunkSource{cms: cms, chunk: chunk, stats: cms.stats}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

var errReqForChunk = errors.New("no request support for chunk sources")

func (cms *chunkMultiSource) SourceFromRequest(context.Context, *api.WorkerRequest) (driver.Source, error) {
	return nil, errReqForChunk
}

func (m *chunkMultiSource) Stats() ChunkStats {
	return m.stats.Copy()
}

type chunkSource struct {
	cms   *chunkMultiSource
	chunk Chunk
	stats *ChunkStats
}

func (s *chunkSource) Open(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter) (driver.ScannerCloser, error) {
	var size int64
	paths := make([]string, len(s.cms.altPaths))
	for i, input := range s.cms.altPaths {
		u := s.chunk.Localize(s.cms.ark, input)
		stat, err := iosrc.Stat(ctx, u)
		if err != nil {
			return nil, err
		}
		size += stat.Size()
		paths[i] = u.String()
	}
	rc := detector.MultiFileReader(zctx, paths, zio.ReaderOpts{Format: "zng"})
	sn, err := scanner.NewScanner(ctx, rc, sf.Filter, sf.FilterExpr, sf.Span)
	if err != nil {
		rc.Close()
		return nil, err
	}
	s.stats.Accumulate(ChunkStats{size, size})
	return &scannerCloser{Scanner: sn, Closer: rc}, nil
}

func (s *chunkSource) ToRequest(*api.WorkerRequest) error {
	return errReqForChunk
}

type ChunkStats struct {
	// ChunksOpenedBytes is the cumulative size of all the chunks read.
	ChunksOpenedBytes int64
	// ChunksReadBytes is the amount of bytes read from all chunks. If seek
	// indicies are used this number should be less than OpenedChunkSize.
	ChunksReadBytes int64
}

func (s *ChunkStats) Accumulate(a ChunkStats) {
	atomic.AddInt64(&s.ChunksOpenedBytes, a.ChunksOpenedBytes)
	atomic.AddInt64(&s.ChunksReadBytes, a.ChunksReadBytes)
}

func (s *ChunkStats) Copy() ChunkStats {
	return ChunkStats{
		ChunksOpenedBytes: atomic.LoadInt64(&s.ChunksOpenedBytes),
		ChunksReadBytes:   atomic.LoadInt64(&s.ChunksReadBytes),
	}
}
