package meta

import (
	"bytes"
	"context"
	"sort"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/errgroup"
)

// Lister enumerates all the data.Objects in a scan.  A Slicer downstream may
// optionally organize objects into non-overlapping partitions for merge on read.
// The optimizer may decide when partitions are necessary based on the order
// sensitivity of the downstream flowgraph.
type Lister struct {
	ctx       context.Context
	pool      *lake.Pool
	snap      commits.View
	filter    zbuf.Filter
	group     *errgroup.Group
	marshaler *zson.MarshalZNGContext
	mu        sync.Mutex
	objects   []*data.Object
	err       error
}

var _ zbuf.Puller = (*Lister)(nil)

func NewSortedLister(ctx context.Context, zctx *zed.Context, r *lake.Root, pool *lake.Pool, commit ksuid.KSUID, filter zbuf.Filter) (*Lister, error) {
	snap, err := pool.Snapshot(ctx, commit)
	if err != nil {
		return nil, err
	}
	return NewSortedListerFromSnap(ctx, zctx, r, pool, snap, filter), nil
}

func NewSortedListerByID(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, filter zbuf.Filter) (*Lister, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return NewSortedLister(ctx, zctx, r, pool, commit, filter)
}

func NewSortedListerFromSnap(ctx context.Context, zctx *zed.Context, r *lake.Root, pool *lake.Pool, snap commits.View, filter zbuf.Filter) *Lister {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return &Lister{
		ctx:       ctx,
		pool:      pool,
		snap:      snap,
		filter:    filter,
		group:     &errgroup.Group{},
		marshaler: m,
	}
}

func (l *Lister) Snapshot() commits.View {
	return l.snap
}

func (l *Lister) Pull(done bool) (zbuf.Batch, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.objects == nil {
		l.objects, l.err = initObjectScan(l.snap, l.pool.Layout, l.filter)
		if l.err != nil {
			return nil, l.err
		}
	}
	// End after the last object.  XXX we could change this so a scan can appear
	// inside of a subgraph and be restarted after each time done is called
	// (like how head/tail work).
	if l.err != nil || len(l.objects) == 0 {
		return nil, l.err
	}
	o := l.objects[0]
	l.objects = l.objects[1:]
	val, err := l.marshaler.Marshal(o)
	if err != nil {
		l.err = err
		return nil, err
	}
	return zbuf.NewArray([]zed.Value{*val}), nil
}

type Slice struct {
	First  *zed.Value
	Last   *zed.Value
	Object *data.Object
}

func initObjectScan(snap commits.View, layout order.Layout, filter zbuf.Filter) ([]*data.Object, error) {
	objects := snap.Select(nil, layout.Order)
	var f *expr.SpanFilter //XXX get rid of this
	if filter != nil {
		var err error
		// Order is passed here just to handle nullsmax in the comparison.
		// So we still need to swap first/last when descending order.
		f, err = filter.AsKeySpanFilter(layout.Primary(), layout.Order)
		if err != nil {
			return nil, err
		}
		if f != nil {
			var filtered []*data.Object
			for _, obj := range objects {
				from := &obj.From
				to := &obj.To
				if layout.Order == order.Desc {
					from, to = to, from
				}
				if !f.Eval(from, to) {
					filtered = append(filtered, obj)
				}
			}
			objects = filtered
		}
	}
	//XXX at some point sorting should be optional.
	sortObjects(objects, layout.Order)
	return objects, nil
}

func sortObjects(objects []*data.Object, o order.Which) {
	cmp := expr.NewValueCompareFn(o, o == order.Asc) //XXX is nullsMax correct here?
	lessFunc := func(a, b *data.Object) bool {
		if cmp(&a.From, &b.From) < 0 {
			return true
		}
		if !bytes.Equal(a.From.Bytes, b.From.Bytes) {
			return false
		}
		if bytes.Equal(a.To.Bytes, b.To.Bytes) {
			// If the pool keys are equal for both the first and last values
			// in the object, we return false here so that the stable sort preserves
			// the commit order of the objects in the log. XXX we might want to
			// simply sort by commit timestamp for a more robust API that does not
			// presume commit-order in the object snapshot.
			return false
		}
		return cmp(&a.To, &b.To) < 0
	}
	sort.SliceStable(objects, func(i, j int) bool {
		return lessFunc(objects[i], objects[j])
	})
}
