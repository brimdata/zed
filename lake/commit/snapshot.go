package commit

import (
	"context"

	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

// A snapshot summarizes the pool state at a given point in the journal.
// XXX TBD: sort log by span and stack snapshot as a base layer
// followed by change sets for each base-layer generation.  This is what
// we will store in the journal sub-pool when this gets implemented.
// See issue #XXX.
// Also, snapshots should have index updates so each segment in the
// snapshot could include which indexes are attached to it.
type Snapshot struct {
	at       journal.ID
	order    zbuf.Order
	segments []segment.Reference
}

func newSnapshot(at journal.ID, order zbuf.Order) *Snapshot {
	return &Snapshot{at: at, order: order}
}

func (s *Snapshot) AddSegment(seg segment.Reference) {
	s.segments = append(s.segments, seg)
}

func (s *Snapshot) DeleteSegment(id ksuid.KSUID) {
	panic("TBD")
}

func (s *Snapshot) Scan(ctx context.Context, ch chan segment.Reference) error {
	for _, seg := range s.segments {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- seg:
		}
	}
	return nil
}

func (s *Snapshot) ScanSpan(ctx context.Context, ch chan<- segment.Reference, span nano.Span) error {
	for _, seg := range s.segments {
		if span.Overlaps(seg.Span()) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- seg:
			}
		}
	}
	return nil
}

func (s *Snapshot) ScanSpanInOrder(ctx context.Context, ch chan<- segment.Reference, span nano.Span) error {
	for _, seg := range s.sortedSegments(span) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- *seg:
		}
	}
	return nil
}

// ScanPartitions partitions all segments in the snapshot overlapping
// with the given span into non-overlapping partitions and sends the
// partitions over the given channel sorted by the pool key in the
// pool's order.
func (s *Snapshot) ScanPartitions(ctx context.Context, ch chan<- segment.Partition, span nano.Span) error {
	segments := s.sortedSegments(span)
	first := span.Ts
	last := span.End()
	if s.order == zbuf.OrderDesc {
		first, last = last, first
	}
	for _, p := range segment.PartitionSegments(segments, s.order) {
		// XXX this is clunky mixing spans and key ranges.
		// When we get rid of the ts assumption, we will fix this.
		// See issue #2482.
		if s.order == zbuf.OrderAsc {
			if p.First < first {
				p.First = first
			}
			if p.Last > last {
				p.Last = last
			}
		} else {
			if p.First > first {
				p.First = first
			}
			if p.Last < last {
				p.Last = last
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- p:
		}
	}
	return nil
}

func (s *Snapshot) sortedSegments(span nano.Span) []*segment.Reference {
	var sorted []*segment.Reference
	for k, seg := range s.segments {
		if span.Overlaps(seg.Span()) {
			sorted = append(sorted, &s.segments[k])
		}
	}
	segment.Sort(s.order, sorted)
	return sorted
}
