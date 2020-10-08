package archive

import (
	"sort"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/segmentio/ksuid"
)

// mergeChunksToSpans takes an unordered set of Chunks with possibly overlapping
// spans, and returns an ordered list of spanInfos, whose spans will be bounded
// by filter, and where each SpanInfo contains one or more Chunks whose data
// falls into the SpanInfo's span.
func mergeChunksToSpans(chunks []Chunk, dir zbuf.Direction, filter nano.Span) []SpanInfo {
	sinfos := alignChunksToSpans(chunks, dir, filter)
	sinfos = mergeLargestChunkSpanInfos(sinfos, dir)
	return sinfos
}

// alignChunksToSpans creates an ordered slice of SpanInfo's whose boundaries
// match either the boundaries of a chunk, or in the case of overlapping chunks,
// the boundaries of each portion of an overlap.
func alignChunksToSpans(chunks []Chunk, dir zbuf.Direction, filter nano.Span) []SpanInfo {
	var siChunks []Chunk // accumulating chunks for next SpanInfo
	var siFirst nano.Ts  // first timestamp for next SpanInfo
	var result []SpanInfo
	boundaries(chunks, dir, func(ts nano.Ts, firstChunks, lastChunks []Chunk) {
		if len(firstChunks) > 0 {
			// ts is the 'First' timestamp for these chunks.
			if len(siChunks) > 0 {
				// We have accumulated chunks; create a span with them whose
				// last timestamp was just before ts.
				siSpan := firstLastToSpan(siFirst, prevTs(ts, dir))
				if filter.Overlaps(siSpan) {
					result = append(result, SpanInfo{
						Span:   filter.Intersect(siSpan),
						Chunks: copyChunks(siChunks, nil),
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
				result = append(result, SpanInfo{
					Span:   filter.Intersect(siSpan),
					Chunks: copyChunks(siChunks, nil),
				})
			}
			// Drop the chunks that ended from our accumulation.
			siChunks = copyChunks(siChunks, lastChunks)
			siFirst = nextTs(ts, dir)
		}
	})
	return result
}

func copyChunks(src []Chunk, skip []Chunk) (dst []Chunk) {
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

func spanToFirstLast(dir zbuf.Direction, span nano.Span) (nano.Ts, nano.Ts) {
	if dir == zbuf.DirTimeForward {
		return span.Ts, span.End() - 1
	}
	return span.End() - 1, span.Ts
}

func nextTs(ts nano.Ts, dir zbuf.Direction) nano.Ts {
	if dir == zbuf.DirTimeForward {
		return ts + 1
	}
	return ts - 1
}

func prevTs(ts nano.Ts, dir zbuf.Direction) nano.Ts {
	if dir == zbuf.DirTimeForward {
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
func boundaries(chunks []Chunk, dir zbuf.Direction, fn func(ts nano.Ts, firstChunks, lastChunks []Chunk)) {
	points := make([]point, 2*len(chunks))
	for i, c := range chunks {
		points[2*i] = point{idx: i, first: true, ts: c.First}
		points[2*i+1] = point{idx: i, ts: c.Last}
	}
	sort.Slice(points, func(i, j int) bool {
		return chunkTsLess(dir, points[i].ts, chunks[points[i].idx].Id, points[j].ts, chunks[points[j].idx].Id)
	})
	firstChunks := make([]Chunk, 0, len(chunks))
	lastChunks := make([]Chunk, 0, len(chunks))
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

func largestChunk(si SpanInfo) Chunk {
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

func mergeSpanInfos(sis []SpanInfo, dir zbuf.Direction) SpanInfo {
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
	chunksSort(dir, res.Chunks)
	return res
}

// mergeLargestChunkSpanInfos merges contiguous SpanInfo's whose largest
// Chunks by RecordCount is the same in the input slice.
func mergeLargestChunkSpanInfos(spans []SpanInfo, dir zbuf.Direction) []SpanInfo {
	if len(spans) < 2 {
		return spans
	}
	var res []SpanInfo
	run := []SpanInfo{spans[0]}
	runLargest := largestChunk(spans[0])
	for i := 1; i < len(spans); i++ {
		largest := largestChunk(spans[i])
		if largest.Id != runLargest.Id {
			res = append(res, mergeSpanInfos(run, dir))
			run = []SpanInfo{spans[i]}
			runLargest = largest
		} else {
			run = append(run, spans[i])
		}
	}
	res = append(res, mergeSpanInfos(run, dir))
	return res
}
