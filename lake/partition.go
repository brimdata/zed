package lake

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

// A Partition is a logical view of the records within a time span, stored
// in one or more Segments.  This provides a way to return the list of
// Segments that should be scanned along with a span to limit the scan
// to only the span involved.
type Partition struct {
	extent.Span
	compare  expr.ValueCompareFn
	Segments []*segment.Reference
}

func (p Partition) IsZero() bool {
	return p.Segments == nil
}

func (p Partition) FormatRangeOf(segno int) string {
	seg := p.Segments[segno]
	return fmt.Sprintf("[%s-%s,%s-%s]", zson.String(p.First()), zson.String(p.Last()), zson.String(seg.First), zson.String(seg.Last))
}

func (p Partition) FormatRange() string {
	return fmt.Sprintf("[%s-%s]", zson.String(p.First()), zson.String(p.Last()))
}

// partitionSegments takes a sorted set of segments with possibly overlapping
// key ranges and returns an ordered list of Ranges such that none of the
// Ranges overlap with one another.  This is the straightforward computational
// geometry problem of merging overlapping intervals,
// e.g., https://www.geeksforgeeks.org/merging-intervals/
//
// XXX this algorithm doesn't quite do what we want because it continues
// to merge *anything* that overlaps.  It's easy to fix though.
// Issue #2538
func PartitionSegments(segments []*segment.Reference, o order.Which) []Partition {
	if len(segments) == 0 {
		return nil
	}
	cmp := extent.CompareFunc(o)
	segspans := sortedSegmentSpans(segments, cmp)
	var s stack
	s.pushSegmentSpan(segspans[0], cmp)
	for _, segspan := range segspans[1:] {
		tos := s.tos()
		if segspan.Before(tos.Last()) {
			s.pushSegmentSpan(segspan, cmp)
		} else {
			tos.Segments = append(tos.Segments, segspan.seg)
			tos.Extend(segspan.Last())
		}
	}
	// On exit, the ranges in the stack are properly sorted so
	// we just return the stack as a []Range.
	return s
}

type stack []Partition

func (s *stack) pushSegmentSpan(segspan segmentSpan, cmp expr.ValueCompareFn) {
	s.push(Partition{
		Span:     segspan.Span,
		compare:  cmp,
		Segments: []*segment.Reference{segspan.seg},
	})
}

func (s *stack) push(p Partition) {
	*s = append(*s, p)
}

func (s *stack) pop() Partition {
	n := len(*s)
	p := (*s)[n-1]
	*s = (*s)[:n-1]
	return p
}

func (s *stack) tos() *Partition {
	return &(*s)[len(*s)-1]
}

type segmentSpan struct {
	extent.Span
	seg *segment.Reference
}

func sortedSegmentSpans(segments []*segment.Reference, cmp expr.ValueCompareFn) []segmentSpan {
	segspans := make([]segmentSpan, 0, len(segments))
	for _, s := range segments {
		segspans = append(segspans, segmentSpan{
			Span: extent.NewGeneric(s.First, s.Last, cmp),
			seg:  s,
		})
	}
	sort.Slice(segspans, func(i, j int) bool {
		return segmentSpanLess(segspans[i], segspans[j])
	})
	return segspans
}

func segmentSpanLess(a, b segmentSpan) bool {
	if b.Before(a.First()) {
		return true
	}
	if !bytes.Equal(a.First().Bytes, b.First().Bytes) {
		return false
	}
	if bytes.Equal(a.Last().Bytes, b.Last().Bytes) {
		if a.seg.Count != b.seg.Count {
			return a.seg.Count < b.seg.Count
		}
		return ksuid.Compare(a.seg.ID, b.seg.ID) < 0
	}
	return a.After(b.Last())
}

func sortSegments(o order.Which, r []*segment.Reference) {
	for k, segSpan := range sortedSegmentSpans(r, extent.CompareFunc(o)) {
		r[k] = segSpan.seg
	}
}
