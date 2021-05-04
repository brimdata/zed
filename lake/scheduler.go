package lake

import (
	"context"
	"sync"

	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

type Scheduler struct {
	ctx    context.Context
	zctx   *zson.Context
	pool   *Pool
	snap   *commit.Snapshot
	span   nano.Span
	filter zbuf.Filter
	once   sync.Once
	ch     chan Partition
	done   chan error
	stats  zbuf.ScannerStats
}

func NewSortedScheduler(ctx context.Context, zctx *zson.Context, pool *Pool, snap *commit.Snapshot, span nano.Span, filter zbuf.Filter) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		zctx:   zctx,
		pool:   pool,
		snap:   snap,
		span:   span,
		filter: filter,
		ch:     make(chan Partition),
		done:   make(chan error),
	}
}

func (s *Scheduler) Stats() zbuf.ScannerStats {
	return s.stats.Copy()
}

func (s *Scheduler) AddStats(stats zbuf.ScannerStats) {
	s.stats.Add(stats)
}

func (s *Scheduler) PullScanTask() (zbuf.PullerCloser, error) {
	s.once.Do(func() {
		go s.run()
	})
	select {
	case p := <-s.ch:
		if p.Segments == nil {
			return nil, <-s.done
		}
		return s.newSortedScanner(p)
	case <-s.ctx.Done():
		return nil, <-s.done
	}
}

func (s *Scheduler) run() {
	if err := ScanPartitions(s.ctx, s.snap, s.span, s.pool.Layout.Order, s.ch); err != nil {
		s.done <- err
	}
	close(s.ch)
	close(s.done)
}

// PullScanWork returns the next span in the schedule.  This is useful for a
// worker proc that pulls spans from teh scheduler, sends them to a remote
// worker, and streams the results into the runtime DAG.
func (s *Scheduler) PullScanWork() (Partition, error) {
	s.once.Do(func() {
		go s.run()
	})
	select {
	case p := <-s.ch:
		return p, nil
	case <-s.ctx.Done():
		return Partition{}, <-s.done
	}
}

func (s *Scheduler) newSortedScanner(p Partition) (zbuf.PullerCloser, error) {
	return newSortedScanner(s.ctx, s.pool, s.zctx, s.filter, p, s)
}

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

func ScanSpanInOrder(ctx context.Context, snap *commit.Snapshot, span nano.Span, o order.Which, ch chan<- segment.Reference) error {
	segments := snap.Select(span)
	segment.Sort(o, segments)
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
func ScanPartitions(ctx context.Context, snap *commit.Snapshot, span nano.Span, o order.Which, ch chan<- Partition) error {
	first := span.Ts
	last := span.End()
	if o == order.Desc {
		first, last = last, first
	}
	segments := snap.Select(span)
	segment.Sort(o, segments)
	for _, p := range PartitionSegments(segments, o) {
		// XXX this is clunky mixing spans and key ranges.
		// When we get rid of the ts assumption, we will fix this.
		// See issue #2482.
		if o == order.Asc {
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
