package exec

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/meta"
	"github.com/brimdata/zed/runtime/op/from"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/errgroup"
)

type Planner struct {
	ctx      context.Context
	zctx     *zed.Context
	pool     *lake.Pool
	snap     commits.View
	filter   zbuf.Filter
	once     sync.Once
	ch       chan meta.Partition
	group    *errgroup.Group
	progress zbuf.Progress
}

var _ from.Planner = (*Planner)(nil)

func NewSortedPlanner(ctx context.Context, zctx *zed.Context, pool *lake.Pool, snap commits.View, filter zbuf.Filter) (*Planner, error) {
	return &Planner{
		ctx:    ctx,
		zctx:   zctx,
		pool:   pool,
		snap:   snap,
		filter: filter,
		ch:     make(chan meta.Partition),
	}, nil
}

func (p *Planner) Progress() zbuf.Progress {
	return p.progress.Copy()
}

func (p *Planner) PullWork() (zbuf.Puller, error) {
	p.once.Do(func() {
		p.run()
	})
	select {
	case part := <-p.ch:
		if part.Objects == nil {
			return nil, p.group.Wait()
		}
		return newPartitionScanner(p, part)
	case <-p.ctx.Done():
		return nil, p.group.Wait()
	}
}

func (p *Planner) run() {
	var ctx context.Context
	p.group, ctx = errgroup.WithContext(p.ctx)
	p.group.Go(func() error {
		defer close(p.ch)
		return ScanPartitions(ctx, p.snap, p.pool.Layout, p.filter, p.ch)
	})
}

// PullScanWork returns the next span in the schedule.  This is useful for a
// worker proc that pulls spans from teh scheduler, sends them to a remote
// worker, and streams the results into the runtime DAG.
func (p *Planner) PullScanWork() (meta.Partition, error) {
	p.once.Do(func() {
		p.run()
	})
	select {
	case part := <-p.ch:
		return part, nil
	case <-p.ctx.Done():
		return meta.Partition{}, p.group.Wait()
	}
}

type scannerScheduler struct {
	scanners []zbuf.Scanner
	progress zbuf.Progress
	last     zbuf.Scanner
}

var _ from.Planner = (*scannerScheduler)(nil)

func newScannerScheduler(scanners ...zbuf.Scanner) *scannerScheduler {
	return &scannerScheduler{
		scanners: scanners,
	}
}

func (s *scannerScheduler) PullWork() (zbuf.Puller, error) {
	if s.last != nil {
		s.progress.Add(s.last.Progress())
		s.last = nil
	}
	if len(s.scanners) > 0 {
		s.last = s.scanners[0]
		s.scanners = s.scanners[1:]
		return s.last, nil
	}
	return nil, nil
}

func (s *scannerScheduler) Progress() zbuf.Progress {
	return s.progress.Copy()
}

func Scan(ctx context.Context, snap commits.View, o order.Which, ch chan<- data.Object) error {
	for _, object := range snap.Select(nil, o) {
		select {
		case ch <- *object:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func ScanInOrder(ctx context.Context, snap commits.View, o order.Which, ch chan<- data.Object) error {
	objects := snap.Select(nil, o)
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

func filterObjects(objects []*data.Object, filter *expr.SpanFilter, o order.Which) []*data.Object {
	cmp := expr.NewValueCompareFn(o == order.Asc)
	out := objects[:0]
	for _, obj := range objects {
		span := extent.NewGeneric(obj.First, obj.Last, cmp)
		if filter == nil || !filter.Eval(span.First(), span.Last()) {
			out = append(out, obj)
		}
	}
	return out
}

// ScanPartitions partitions all the data objects in snap overlapping
// span into non-overlapping partitions, sorts them by pool key and order,
// and sends them to ch.
func ScanPartitions(ctx context.Context, snap commits.View, layout order.Layout, filter zbuf.Filter, ch chan<- meta.Partition) error {
	objects := snap.Select(nil, layout.Order)
	f, err := filter.AsKeySpanFilter(layout.Primary(), layout.Order)
	if err != nil {
		return err
	}
	objects = filterObjects(objects, f, layout.Order)
	for _, p := range PartitionObjects(objects, layout.Order) {
		select {
		case ch <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func ScanIndexes(ctx context.Context, snap commits.View, o order.Which, ch chan<- *index.Object) error {
	for _, idx := range snap.SelectIndexes(nil, o) {
		select {
		case ch <- idx:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func NewPlanner(ctx context.Context, zctx *zed.Context, p *lake.Pool, commit ksuid.KSUID, filter zbuf.Filter) (from.Planner, error) {
	snap, err := p.Snapshot(ctx, commit)
	if err != nil {
		return nil, err
	}
	return NewSortedPlanner(ctx, zctx, p, snap, filter)
}

func NewPlannerByID(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, filter zbuf.Filter) (from.Planner, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return NewPlanner(ctx, zctx, pool, commit, filter)
}
