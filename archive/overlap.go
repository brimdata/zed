package archive

import (
	"sort"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

// mergeChunksToSpans takes a set of Chunks with possibly overlapping spans,
// and returns an ordered list of SpanInfos, whose spans will be bounded by
// filter, and where each SpanInfo contains one or more Chunks whose data
// falls into the SpanInfo's span.
func mergeChunksToSpans(chunks []Chunk, dir zbuf.Direction, filter nano.Span) []SpanInfo {
	var siChunks []Chunk // accumulating chunks for next SpanInfo
	var siFirst nano.Ts  // first timestamp for next SpanInfo
	var result []SpanInfo
	boundaries(chunks, dir, func(ts nano.Ts, firstChunks, lastChunks []Chunk) {
		if len(firstChunks) > 0 {
			// ts is the 'First' timestamp for these chunks.
			if len(siChunks) > 0 {
				// We have accumulated chunks; create a span with them whose
				// last timestamp was just before ts.
				siSpan := closedSpan(siFirst, prevTs(ts, dir))
				if filter.Overlaps(siSpan) {
					first, last := spanToFirstLast(dir, filter.Intersect(siSpan))
					result = append(result, SpanInfo{
						First:  first,
						Last:   last,
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
			siSpan := closedSpan(siFirst, ts)
			if filter.Overlaps(siSpan) {
				first, last := spanToFirstLast(dir, filter.Intersect(siSpan))
				result = append(result, SpanInfo{
					First:  first,
					Last:   last,
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

// closedSpan returns a span for the closed interval of [x,y].
func closedSpan(x, y nano.Ts) nano.Span {
	return nano.Span{Ts: x, Dur: 1}.Union(nano.Span{Ts: y, Dur: 1})
}

// spanToFirstLast returns the timestamps that whose closed interval
// is represented by span s. It assumes s.Dur is greater than zero.
func spanToFirstLast(dir zbuf.Direction, s nano.Span) (first, last nano.Ts) {
	a := s.Ts
	b := s.Ts + nano.Ts(s.Dur-1)
	if a > b {
		a, b = b, a
	}
	if dir == zbuf.DirTimeForward {
		return a, b
	}
	return b, a
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
	c     Chunk
	first bool
	ts    nano.Ts
}

// boundaries sorts the given chunks, then calls fn with each timestamp that
// acts as a first and/or last timestamp of one or more of the chunks.
func boundaries(chunks []Chunk, dir zbuf.Direction, fn func(ts nano.Ts, firstChunks, lastChunks []Chunk)) {
	points := make([]point, 2*len(chunks))
	for i, c := range chunks {
		points[2*i] = point{c: c, first: true, ts: c.First}
		points[2*i+1] = point{c: c, ts: c.Last}
	}
	sort.Slice(points, func(i, j int) bool {
		if dir == zbuf.DirTimeForward {
			return points[i].ts < points[j].ts
		}
		return points[j].ts < points[i].ts
	})
	var firstChunks, lastChunks []Chunk
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
				firstChunks = append(firstChunks, p.c)
			} else {
				lastChunks = append(lastChunks, p.c)
			}
		}
		ts := points[i].ts
		i = j
		fn(ts, firstChunks, lastChunks)
	}
}
