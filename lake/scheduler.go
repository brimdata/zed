package lake

import (
	"context"
	"errors"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"golang.org/x/sync/errgroup"
)

type Scheduler struct {
	ctx      context.Context
	zctx     *zed.Context
	pool     *Pool
	snap     commits.View
	span     extent.Span
	filter   zbuf.Filter
	index    *index.Filter
	once     sync.Once
	ch       chan Partition
	group    *errgroup.Group
	progress zbuf.Progress
}

var _ op.Scheduler = (*Scheduler)(nil)

func NewSortedScheduler(ctx context.Context, zctx *zed.Context, pool *Pool, snap commits.View, span extent.Span, filter zbuf.Filter, index *index.Filter) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		zctx:   zctx,
		pool:   pool,
		snap:   snap,
		span:   span,
		filter: filter,
		index:  index,
		ch:     make(chan Partition),
	}
}

func (s *Scheduler) Progress() zbuf.Progress {
	return s.progress.Copy()
}

func (s *Scheduler) AddProgress(progress zbuf.Progress) {
	s.progress.Add(progress)
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
		// Make the index out channel buffered so the index filter reads ahead
		// of the scan pass, but not too far ahead- the search may be aborted
		// before the scan pass gets to the end and we don't want to waste
		// resources running index lookups that aren't used.
		ch = make(chan Partition)
		s.group.Go(func() error {
			defer close(s.ch)
			return indexFilterPass(ctx, s.pool, s.snap, s.index, ch, s.ch)
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
	progress zbuf.Progress
	last     zbuf.Scanner
}

var _ op.Scheduler = (*scannerScheduler)(nil)

func newScannerScheduler(scanners ...zbuf.Scanner) *scannerScheduler {
	return &scannerScheduler{
		scanners: scanners,
	}
}

func (s *scannerScheduler) PullScanTask() (zbuf.PullerCloser, error) {
	if s.last != nil {
		s.progress.Add(s.last.Progress())
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

func (s *scannerScheduler) Progress() zbuf.Progress {
	return s.progress.Copy()
}

func indexFilterPass(ctx context.Context, pool *Pool, snap commits.View, filter *index.Filter, in <-chan Partition, out chan<- Partition) error {
	for p := range in {
		objects := make([]*data.ObjectScan, 0, len(p.Objects))
		for _, o := range p.Objects {
			r, err := snap.LookupIndexObjectRules(o.ID)
			if err != nil && !errors.Is(err, commits.ErrNotFound) {
				return err
			}
			if r != nil {
				span, err := filter.Apply(ctx, o.ID, r)
				if err != nil {
					return err
				}
				if span == nil {
					continue
				}
				if err := seekIndexByCount(ctx, pool, o, span); err != nil {
					return err
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

func seekIndexByCount(ctx context.Context, pool *Pool, o *data.ObjectScan, span extent.Span) error {
	r, err := pool.engine.Get(ctx, o.SeekObjectPath(pool.DataPath))
	if err != nil {
		return err

	}
	defer r.Close()
	zr := zngio.NewReader(r, zed.NewContext())
	rg, err := seekindex.LookupByCount(zr, span.First(), span.Last())
	if err != nil {
		return err
	}
	o.ScanRange = rg
	return nil
}

func ScanSpan(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- data.Object) error {
	for _, object := range snap.Select(span, o) {
		if span == nil || span.Overlaps(&object.First, &object.Last) {
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
