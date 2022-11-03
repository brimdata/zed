package meta

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
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/errgroup"
)

// XXX Lister enumerates all the partitions in a scan.  After we get this working,
// we will modularize further by listing just the objects and having
// the slicer to organize objects into slices (formerly known as partitions).
type Lister struct {
	ctx       context.Context
	pool      *lake.Pool
	snap      commits.View
	filter    zbuf.Filter
	once      sync.Once
	ch        chan Partition
	group     *errgroup.Group
	marshaler *zson.MarshalZNGContext
	mu        sync.Mutex
}

var _ zbuf.Puller = (*Lister)(nil)

func NewSortedLister(ctx context.Context, r *lake.Root, pool *lake.Pool, commit ksuid.KSUID, filter zbuf.Filter) (*Lister, error) {
	snap, err := pool.Snapshot(ctx, commit)
	if err != nil {
		return nil, err
	}
	return NewSortedListerFromSnap(ctx, r, pool, snap, filter), nil
}

func NewSortedListerByID(ctx context.Context, r *lake.Root, poolID, commit ksuid.KSUID, filter zbuf.Filter) (*Lister, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return NewSortedLister(ctx, r, pool, commit, filter)
}

func NewSortedListerFromSnap(ctx context.Context, r *lake.Root, pool *lake.Pool, snap commits.View, filter zbuf.Filter) *Lister {
	return &Lister{
		ctx:       ctx,
		pool:      pool,
		snap:      snap,
		filter:    filter,
		once:      sync.Once{},
		ch:        make(chan Partition), //XXX
		group:     &errgroup.Group{},
		marshaler: zson.NewZNGMarshaler(),
	}
}

func (l *Lister) Snapshot() commits.View {
	return l.snap
}

func (l *Lister) Pull(done bool) (zbuf.Batch, error) {
	l.once.Do(func() {
		l.run()
	})
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case part := <-l.ch:
		if part.Objects == nil {
			err := l.group.Wait()
			return nil, err
		}
		val, err := l.marshaler.Marshal(part)
		if err != nil {
			return nil, err
		}
		return zbuf.NewArray([]zed.Value{*val}), nil
	case <-l.ctx.Done():
		return nil, l.group.Wait()
	}
}

func (l *Lister) run() {
	var ctx context.Context
	l.group, ctx = errgroup.WithContext(l.ctx)
	l.group.Go(func() error {
		defer close(l.ch)
		return ScanPartitions(ctx, l.snap, l.pool.Layout, l.filter, l.ch)
	})
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
func ScanPartitions(ctx context.Context, snap commits.View, layout order.Layout, filter zbuf.Filter, ch chan<- Partition) error {
	objects := snap.Select(nil, layout.Order)
	if filter != nil {
		f, err := filter.AsKeySpanFilter(layout.Primary(), layout.Order)
		if err != nil {
			return err
		}
		objects = filterObjects(objects, f, layout.Order)
	}
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
