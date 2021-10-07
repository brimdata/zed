package lake

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"golang.org/x/sync/errgroup"
)

type Scheduler struct {
	ctx    context.Context
	zctx   *zed.Context
	pool   *Pool
	snap   commits.View
	span   extent.Span
	filter zbuf.Filter
	index  *index.Filter
	once   sync.Once
	ch     chan Partition
	group  *errgroup.Group
	stats  zbuf.ScannerStats
}

var _ proc.Scheduler = (*Scheduler)(nil)

func NewSortedScheduler(ctx context.Context, zctx *zed.Context, pool *Pool, snap commits.View, span extent.Span, filter zbuf.Filter, index *index.Filter) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		zctx:   zctx,
		pool:   pool,
		snap:   snap,
		span:   span,
		filter: filter,
		index:  index,
		ch:     make(chan Partition, 10),
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
		s.run()
	})
	select {
	case p := <-s.ch:
		if p.Objects == nil {
			return nil, s.group.Wait()
		}
		return s.newSortedScanner(p)
	case <-s.ctx.Done():
		return nil, s.group.Wait()
	}
}

func (s *Scheduler) run() {
	var ctx context.Context
	s.group, ctx = errgroup.WithContext(s.ctx)
	ch := s.ch
	if s.index != nil {
		ch = make(chan Partition)
		s.group.Go(func() error {
			defer close(s.ch)
			return indexFilterPass(ctx, s.snap, s.index, ch, s.ch)
		})
	}
	s.group.Go(func() error {
		defer close(ch)
		return ScanPartitions(ctx, s.snap, s.span, s.pool.Layout.Order, ch)
	})
}

// PullScanWork returns the next span in the schedule.  This is useful for a
// worker proc that pulls spans from teh scheduler, sends them to a remote
// worker, and streams the results into the runtime DAG.
func (s *Scheduler) PullScanWork() (Partition, error) {
	s.once.Do(func() {
		s.run()
	})
	select {
	case p := <-s.ch:
		return p, nil
	case <-s.ctx.Done():
		return Partition{}, s.group.Wait()
	}
}

func (s *Scheduler) newSortedScanner(p Partition) (zbuf.PullerCloser, error) {
	return newSortedScanner(s.ctx, s.pool, s.zctx, s.filter, p, s)
}

type scannerScheduler struct {
	scanners []zbuf.Scanner
	stats    zbuf.ScannerStats
	last     zbuf.Scanner
}

var _ proc.Scheduler = (*scannerScheduler)(nil)

func newScannerScheduler(scanners ...zbuf.Scanner) *scannerScheduler {
	return &scannerScheduler{
		scanners: scanners,
	}
}

func (s *scannerScheduler) PullScanTask() (zbuf.PullerCloser, error) {
	if s.last != nil {
		s.stats.Add(s.last.Stats())
		s.last = nil
	}
	if len(s.scanners) > 0 {
		scanner := s.scanners[0]
		s.scanners = s.scanners[1:]
		s.last = scanner
		return zbuf.ScannerNopCloser(scanner), nil
	}
	return nil, nil
}

func (s *scannerScheduler) Stats() zbuf.ScannerStats {
	return s.stats.Copy()
}

func indexFilterPass(ctx context.Context, snap commits.View, filter *index.Filter, in <-chan Partition, out chan<- Partition) error {
	for p := range in {
		objects := make([]*data.Object, 0, len(p.Objects))
		for _, o := range p.Objects {
			r, _ := snap.LookupIndexObjectRules(o.ID)
			if r != nil {
				hit, err := filter.Apply(ctx, o.ID, r)
				if err != nil {
					return err
				}
				if !hit {
					continue
				}
			}
			objects = append(objects, o)
		}
		if len(objects) > 0 {
			p.Objects = objects
			select {
			case out <- p:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func ScanSpan(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- data.Object) error {
	for _, object := range snap.Select(span, o) {
		if span == nil || span.Overlaps(object.First, object.Last) {
			select {
			case ch <- *object:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func ScanSpanInOrder(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- data.Object) error {
	objects := snap.Select(span, o)
	sortObjects(o, objects)
	for _, object := range objects {
		select {
		case ch <- *object:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// ScanPartitions partitions all the data objects in snap overlapping
// span into non-overlapping partitions, sorts them by pool key and order,
// and sends them to ch.
func ScanPartitions(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- Partition) error {
	objects := snap.Select(span, o)
	for _, p := range PartitionObjects(objects, o) {
		if span != nil {
			p.Span.Crop(span)
		}
		select {
		case ch <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func ScanIndexes(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- *index.Object) error {
	for _, idx := range snap.SelectIndexes(span, o) {
		select {
		case ch <- idx:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
