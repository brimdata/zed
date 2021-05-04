package lake

import (
	"fmt"

	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
)

// A Partition is a logical view of the records within a time span, stored
// in one or more Segments.  This provides a way to return the list of
// Segments that should be scanned along with a span to limit the scan
// to only the span involved.
// XXX need to change Span to key range.
type Partition struct {
	First    nano.Ts //XXX should be key range
	Last     nano.Ts
	Segments []*segment.Reference
}

func (p Partition) IsZero() bool {
	return p.Segments == nil
}

// Span returns a span that includes the ranges First and Last values
// independent of order.
func (p Partition) Span() nano.Span {
	return nano.Span{Ts: p.First, Dur: 1}.Union(nano.Span{Ts: p.Last, Dur: 1})
}

func (p Partition) FormatRangeOf(segno int) string {
	seg := p.Segments[segno]
	return fmt.Sprintf("[%d-%d,%d-%d]", p.First, p.Last, seg.First, seg.Last)
}

func (p Partition) FormatRange() string {
	return fmt.Sprintf("[%d-%d]", p.First, p.Last)
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
	var s stack
	s.pushSegment(segments[0])
	for _, seg := range segments[1:] {
		tos := s.tos()
		if o == order.Asc {
			if tos.Last < seg.First {
				s.pushSegment(seg)
			} else {
				tos.Segments = append(tos.Segments, seg)
				if tos.Last < seg.Last {
					tos.Last = seg.Last
				}
			}
		} else {
			if tos.Last > seg.First {
				s.pushSegment(seg)
			} else {
				tos.Segments = append(tos.Segments, seg)
				if tos.Last > seg.Last {
					tos.Last = seg.Last
				}
			}
		}
	}
	// On exit, the ranges in the stack are properly sorted so
	// we just return the stack as a []Range.
	return s
}

type stack []Partition

func (s *stack) pushSegment(seg *segment.Reference) {
	s.push(Partition{
		First:    seg.First,
		Last:     seg.Last,
		Segments: []*segment.Reference{seg},
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
