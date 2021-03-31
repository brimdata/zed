package lake

import (
	"context"
	"sort"

	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake/chunk"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

// mergeChunksToSpans takes an unordered set of Chunks with possibly overlapping
// spans, and returns an ordered list of spanInfos, whose spans will be bounded
// by filter, and where each SpanInfo contains one or more Chunks whose data
// falls into the SpanInfo's span.
func mergeChunksToSpans(chunks []chunk.Chunk, order zbuf.Order, filter nano.Span) []SpanInfo {
	sinfos := alignChunksToSpans(chunks, order, filter)
	removeMaskedChunks(sinfos, false)
	return mergeLargestChunkSpanInfos(sinfos, order)
}

// alignChunksToSpans creates an ordered slice of SpanInfo's whose boundaries
// match either the boundaries of a chunk or, in the case of overlapping chunks,
// the boundaries of each portion of an overlap.
func alignChunksToSpans(chunks []chunk.Chunk, order zbuf.Order, filter nano.Span) []SpanInfo {
	var siChunks []chunk.Chunk // accumulating chunks for next SpanInfo
	var siFirst nano.Ts        // first timestamp for next SpanInfo
	var result []SpanInfo
	boundaries(chunks, order, func(ts nano.Ts, firstChunks, lastChunks []chunk.Chunk) {
		if len(firstChunks) > 0 {
			// ts is the 'First' timestamp for these chunks.
			if len(siChunks) > 0 && ts != siFirst {
				// We have accumulated chunks; create a span with them whose
				// last timestamp was just before ts.
				siSpan := firstLastToSpan(siFirst, prevTs(ts, order))
				if filter.Overlaps(siSpan) {
					chunks := copyChunks(siChunks, nil)
					chunk.Sort(order, chunks)
					result = append(result, SpanInfo{
						Span:   filter.Intersect(siSpan),
						Chunks: chunks,
					})
				}
			}
			// Accumulate these chunks whose first timestamp is ts.
			siChunks = append(siChunks, firstChunks...)
			siFirst = ts
		}
		if len(lastChunks) > 0 {
			// ts is the 'Last' timestamp for these chunks.
			siSpan := firstLastToSpan(siFirst, ts)
			if filter.Overlaps(siSpan) {
				chunks := copyChunks(siChunks, nil)
				chunk.Sort(order, chunks)
				result = append(result, SpanInfo{
					Span:   filter.Intersect(siSpan),
					Chunks: chunks,
				})
			}
			// Drop the chunks that ended from our accumulation.
			siChunks = copyChunks(siChunks, lastChunks)
			siFirst = nextTs(ts, order)
		}
	})
	return result
}

func copyChunks(src []chunk.Chunk, skip []chunk.Chunk) (dst []chunk.Chunk) {
outer:
	for i := range src {
		for j := range skip {
			if src[i].Id == skip[j].Id {
				continue outer
			}
		}
		dst = append(dst, src[i])
	}
	return
}

// firstLastToSpan returns a span that includes x and y and does not require
// them to be in any order.
func firstLastToSpan(x, y nano.Ts) nano.Span {
	return nano.Span{Ts: x, Dur: 1}.Union(nano.Span{Ts: y, Dur: 1})
}

func spanToFirstLast(order zbuf.Order, span nano.Span) (nano.Ts, nano.Ts) {
	if order == zbuf.OrderAsc {
		return span.Ts, span.End() - 1
	}
	return span.End() - 1, span.Ts
}

func nextTs(ts nano.Ts, order zbuf.Order) nano.Ts {
	if order == zbuf.OrderAsc {
		return ts + 1
	}
	return ts - 1
}

func prevTs(ts nano.Ts, order zbuf.Order) nano.Ts {
	if order == zbuf.OrderAsc {
		return ts - 1
	}
	return ts + 1
}

type point struct {
	idx   int
	first bool
	ts    nano.Ts
}

// boundaries sorts the given chunks, then calls fn with each timestamp that
// acts as a first and/or last timestamp of one or more of the chunks.
func boundaries(chunks []chunk.Chunk, order zbuf.Order, fn func(ts nano.Ts, firstChunks, lastChunks []chunk.Chunk)) {
	points := make([]point, 2*len(chunks))
	for i, c := range chunks {
		points[2*i] = point{idx: i, first: true, ts: c.First}
		points[2*i+1] = point{idx: i, ts: c.Last}
	}
	sort.Slice(points, func(i, j int) bool {
		if order == zbuf.OrderAsc {
			return points[i].ts < points[j].ts
		}
		return points[j].ts < points[i].ts
	})
	firstChunks := make([]chunk.Chunk, 0, len(chunks))
	lastChunks := make([]chunk.Chunk, 0, len(chunks))
	for i := 0; i < len(points); {
		j := i + 1
		for ; j < len(points); j++ {
			if points[i].ts != points[j].ts {
				break
			}
		}
		firstChunks = firstChunks[:0]
		lastChunks = lastChunks[:0]
		for _, p := range points[i:j] {
			if p.first {
				firstChunks = append(firstChunks, chunks[p.idx])
			} else {
				lastChunks = append(lastChunks, chunks[p.idx])
			}
		}
		ts := points[i].ts
		i = j
		fn(ts, firstChunks, lastChunks)
	}
}

func largestChunk(si SpanInfo) chunk.Chunk {
	res := si.Chunks[0]
	for _, c := range si.Chunks {
		if c.RecordCount > res.RecordCount {
			res = c
		}
	}
	return res
}

func spanInfoContainsChunk(si SpanInfo, cid ksuid.KSUID) bool {
	for _, c := range si.Chunks {
		if c.Id == cid {
			return true
		}
	}
	return false
}

func mergeSpanInfos(sis []SpanInfo, order zbuf.Order) SpanInfo {
	if len(sis) == 1 {
		return sis[0]
	}
	var res SpanInfo
	res.Span = sis[0].Span
	for _, si := range sis {
		res.Span = res.Span.Union(si.Span)
		for _, c := range si.Chunks {
			if !spanInfoContainsChunk(res, c.Id) {
				res.Chunks = append(res.Chunks, c)
			}
		}
	}
	chunk.Sort(order, res.Chunks)
	return res
}

// mergeLargestChunkSpanInfos merges contiguous SpanInfo's whose largest
// Chunk by RecordCount is the same.
func mergeLargestChunkSpanInfos(spans []SpanInfo, order zbuf.Order) []SpanInfo {
	if len(spans) < 2 {
		return spans
	}
	var res []SpanInfo
	run := []SpanInfo{spans[0]}
	runLargest := largestChunk(spans[0])
	for _, s := range spans[1:] {
		largest := largestChunk(s)
		if largest.Id != runLargest.Id {
			res = append(res, mergeSpanInfos(run, order))
			run = []SpanInfo{s}
			runLargest = largest
		} else {
			run = append(run, s)
		}
	}
	return append(res, mergeSpanInfos(run, order))
}

// removeMaskedChunks performs an in-place edit of the spans input slice,
// updating each SpanInfo to remove chunks that mask other chunks within the
// same SpanInfo.
// If trackMasked is true, it will return a slice of chunks that have been
// masked and are no longer present in any SpanInfo.
func removeMaskedChunks(spans []SpanInfo, trackMasked bool) []chunk.Chunk {
	var maskedChunks map[ksuid.KSUID]chunk.Chunk
	for i := range spans {
		rem := spans[i].RemoveMasked()
		if trackMasked && len(rem) > 0 {
			if maskedChunks == nil {
				maskedChunks = make(map[ksuid.KSUID]chunk.Chunk)
			}
			for _, c := range rem {
				maskedChunks[c.Id] = c
			}
		}
	}
	var mc []chunk.Chunk
	if trackMasked {
	outer:
		for id := range maskedChunks {
			// This check ensures that only chunks that are completely masked
			// within the input []SpanInfo are returned.
			for _, si := range spans {
				if spanInfoContainsChunk(si, id) {
					continue outer
				}
			}
			mc = append(mc, maskedChunks[id])
		}
	}
	return mc
}

type compactWriter struct {
	lk      *Lake
	ctx     context.Context
	created []chunk.Chunk
	defs    index.Definitions
	masks   []ksuid.KSUID
	tsd     tsDir
	w       *chunk.Writer
}

func (cw *compactWriter) Write(rec *zng.Record) error {
	if cw.w != nil {
		pos, firstTs, lastTs := cw.w.Position()
		if pos > cw.lk.LogSizeThreshold && lastTs != rec.Ts() {
			// If we need to create a new chunk writer, we must ensure that the
			// span for the current chunk leaves no gap between its last timestamp
			// and the first timestamp of our next chunk, to ensure these chunks
			// are seen as covering the entire span of the source chunks. Hence
			// the use of 'chunkLastTs', and the lastTs check above to ensure we are
			// not in a run of records with the same timestamp.
			chunkLastTs := prevTs(rec.Ts(), cw.lk.DataOrder)
			if err := cw.w.CloseWithTs(cw.ctx, firstTs, chunkLastTs); err != nil {
				return err
			}
			cw.created = append(cw.created, cw.w.Chunk())
			cw.w = nil
		}
	}
	if cw.w == nil {
		var err error
		cw.w, err = chunk.NewWriter(cw.ctx, cw.tsd.path(cw.lk), chunk.WriterOpts{
			Order:       cw.lk.DataOrder,
			Masks:       cw.masks,
			Definitions: cw.defs,
			Zng: zngio.WriterOpts{
				StreamRecordsMax: ImportStreamRecordsMax,
				LZ4BlockSize:     importLZ4BlockSize,
			},
		})
		if err != nil {
			return err
		}
	}
	return cw.w.Write(rec)
}

// chunks returns the Chunks written by the compact writer. This is only valid
// after close() has returned a nil error.
func (cw *compactWriter) chunks() []chunk.Chunk {
	return cw.created
}

func (cw *compactWriter) abort() {
	if cw.w != nil {
		cw.w.Abort()
		cw.w = nil
	}
}

func (cw *compactWriter) close(lastTs nano.Ts) error {
	if cw.w == nil {
		return nil
	}
	_, firstTs, _ := cw.w.Position()
	if err := cw.w.CloseWithTs(cw.ctx, firstTs, lastTs); err != nil {
		return err
	}
	cw.w = nil
	return nil
}

func compactOverlaps(ctx context.Context, lk *Lake, s SpanInfo) ([]chunk.Chunk, error) {
	if len(s.Chunks) == 1 {
		return nil, nil
	}
	ss, _, err := newSpanScanner(ctx, lk, resolver.NewContext(), driver.SourceFilter{Span: nano.MaxSpan}, s)
	if err != nil {
		return nil, err
	}
	defer ss.Close()
	var masks []ksuid.KSUID
	for _, c := range s.Chunks {
		masks = append(masks, c.Id)
	}
	defs, err := lk.ReadDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	mw := &compactWriter{
		lk:    lk,
		ctx:   ctx,
		defs:  defs,
		masks: masks,
		tsd:   newTsDir(s.Span.Ts),
	}
	_, slast := spanToFirstLast(lk.DataOrder, s.Span)
	if err := zbuf.CopyWithContext(ctx, mw, zbuf.PullerReader(ss)); err != nil {
		mw.abort()
		return nil, err
	}
	if err := mw.close(slast); err != nil {
		return nil, err
	}
	return mw.chunks(), nil
}

// mergeCommonChunkSpans generates a SpanInfo slice by merging runs of
// SpanInfos where a SpanInfo has some chunk in common with its previous
// SpanInfo.
func mergeCommonChunkSpans(spans []SpanInfo, order zbuf.Order) []SpanInfo {
	if len(spans) < 2 {
		return spans
	}
	var res []SpanInfo
	run := []SpanInfo{spans[0]}
outer:
	for _, s := range spans[1:] {
		for _, c := range s.Chunks {
			if spanInfoContainsChunk(run[len(run)-1], c.Id) {
				run = append(run, s)
				continue outer
			}
		}
		res = append(res, mergeSpanInfos(run, order))
		run = []SpanInfo{s}
	}
	return append(res, mergeSpanInfos(run, order))
}

func Compact(ctx context.Context, lk *Lake, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	return tsDirVisit(ctx, lk, nano.MaxSpan, func(_ tsDir, chunks []chunk.Chunk) error {
		spans := alignChunksToSpans(chunks, lk.DataOrder, nano.MaxSpan)
		removeMaskedChunks(spans, false)
		spans = mergeCommonChunkSpans(spans, lk.DataOrder)
		for _, s := range spans {
			newchunks, err := compactOverlaps(ctx, lk, s)
			if err != nil {
				return err
			}
			for _, c := range newchunks {
				m := make([]string, len(c.Masks))
				for i, u := range c.MaskedPaths() {
					m[i] = u.String()
				}

				logger.Info("Compacted chunk created",
					zap.String("chunk", c.Path().String()),
					zap.Strings("masks", m),
				)
			}
		}
		return nil
	})
}

func Purge(ctx context.Context, lk *Lake, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	return tsDirVisit(ctx, lk, nano.MaxSpan, func(_ tsDir, chunks []chunk.Chunk) error {
		spans := alignChunksToSpans(chunks, lk.DataOrder, nano.MaxSpan)
		maskedChunks := removeMaskedChunks(spans, true)
		if len(maskedChunks) == 0 {
			return nil
		}
		for _, c := range maskedChunks {
			if err := c.Remove(ctx); err != nil {
				return err
			}

			logger.Info("Masked chunk purged", zap.String("chunk", c.Path().String()))
		}
		return nil
	})
}
