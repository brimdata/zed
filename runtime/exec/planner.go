package exec

import (
	"context"
	"errors"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/meta"
	"github.com/brimdata/zed/runtime/op/from"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/errgroup"
)

type Planner struct {
	ctx      context.Context
	zctx     *zed.Context
	pool     *lake.Pool
	snap     commits.View
	span     extent.Span
	filter   zbuf.Filter
	index    *index.Filter
	once     sync.Once
	ch       chan meta.Partition
	group    *errgroup.Group
	progress zbuf.Progress
}

var _ from.Planner = (*Planner)(nil)

func NewSortedPlanner(ctx context.Context, zctx *zed.Context, pool *lake.Pool, snap commits.View, span extent.Span, filter zbuf.Filter) (*Planner, error) {
	var idx *index.Filter
	if pd := filter.Pushdown(); pd != nil {
		idx = index.NewFilter(pool.Storage(), pool.IndexPath, pd)
	}
	return &Planner{
		ctx:    ctx,
		zctx:   zctx,
		pool:   pool,
		snap:   snap,
		span:   span,
		filter: filter,
		index:  idx,
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
		return newSortedScanner(p, part)
	case <-p.ctx.Done():
		return nil, p.group.Wait()
	}
}

func (p *Planner) run() {
	var ctx context.Context
	p.group, ctx = errgroup.WithContext(p.ctx)
	ch := p.ch
	if p.index != nil {
		// Make the index out channel buffered so the index filter reads ahead
		// of the scan pass, but not too far ahead- the search may be aborted
		// before the scan pass gets to the end and we don't want to waste
		// resources running index lookups that aren't used.
		ch = make(chan meta.Partition)
		p.group.Go(func() error {
			defer close(p.ch)
			return indexFilterPass(ctx, p.pool, p.snap, p.index, ch, p.ch)
		})
	}
	p.group.Go(func() error {
		defer close(ch)
		return ScanPartitions(ctx, p.snap, p.span, p.pool.Layout.Order, ch)
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

func indexFilterPass(ctx context.Context, pool *lake.Pool, snap commits.View, filter *index.Filter, in <-chan meta.Partition, out chan<- meta.Partition) error {
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

func seekIndexByCount(ctx context.Context, pool *lake.Pool, o *data.ObjectScan, span extent.Span) error {
	r, err := pool.Storage().Get(ctx, o.SeekIndexURI(pool.DataPath))
	if err != nil {
		return err

	}
	defer r.Close()
	zr := zngio.NewReader(r, zed.NewContext())
	defer zr.Close()
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
func ScanPartitions(ctx context.Context, snap commits.View, span extent.Span, o order.Which, ch chan<- meta.Partition) error {
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

func NewPlanner(ctx context.Context, zctx *zed.Context, p *lake.Pool, commit ksuid.KSUID, span extent.Span, filter zbuf.Filter) (from.Planner, error) {
	snap, err := p.Snapshot(ctx, commit)
	if err != nil {
		return nil, err
	}
	return NewSortedPlanner(ctx, zctx, p, snap, span, filter)
}

func NewPlannerByID(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, span extent.Span, filter zbuf.Filter) (from.Planner, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return NewPlanner(ctx, zctx, pool, commit, span, filter)
}
