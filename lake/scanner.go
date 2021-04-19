package lake

import (
	"context"

	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
)

func ScanSpan(ctx context.Context, snap *commit.Snapshot, span nano.Span, ch chan<- segment.Reference) error {
	for _, seg := range snap.Select(span) {
		if span.Overlaps(seg.Span()) {
			select {
			case ch <- *seg:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func ScanSpanInOrder(ctx context.Context, snap *commit.Snapshot, span nano.Span, order zbuf.Order, ch chan<- segment.Reference) error {
	segments := snap.Select(span)
	segment.Sort(order, segments)
	for _, seg := range segments {
		select {
		case ch <- *seg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// ScanPartitions partitions all segments in snap overlapping
// span into non-overlapping partitions, sorts them by pool key and order,
// and sends them to ch.
func ScanPartitions(ctx context.Context, snap *commit.Snapshot, span nano.Span, order zbuf.Order, ch chan<- Partition) error {
	first := span.Ts
	last := span.End()
	if order == zbuf.OrderDesc {
		first, last = last, first
	}
	segments := snap.Select(span)
	segment.Sort(order, segments)
	for _, p := range PartitionSegments(segments, order) {
		// XXX this is clunky mixing spans and key ranges.
		// When we get rid of the ts assumption, we will fix this.
		// See issue #2482.
		if order == zbuf.OrderAsc {
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
		case ch <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
