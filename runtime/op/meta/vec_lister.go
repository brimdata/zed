package meta

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/errgroup"
)

// VecLister enumerates all the data.Objects in a scan.  A Slicer downstream may
// optionally organize objects into non-overlapping partitions for merge on read.
// The optimizer may decide when partitions are necessary based on the order
// sensitivity of the downstream flowgraph.
type VecLister struct {
	ctx       context.Context
	pool      *lake.Pool
	snap      commits.View
	group     *errgroup.Group
	marshaler *zson.MarshalZNGContext
	mu        sync.Mutex
	vectors   []ksuid.KSUID
	err       error
}

var _ zbuf.Puller = (*VecLister)(nil)

func NewVecLister(ctx context.Context, zctx *zed.Context, r *lake.Root, pool *lake.Pool, commit ksuid.KSUID) (*VecLister, error) {
	snap, err := pool.Snapshot(ctx, commit)
	if err != nil {
		return nil, err
	}
	return NewVecListerFromSnap(ctx, zctx, r, pool, snap), nil
}

func NewVecListerByID(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID) (*VecLister, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return NewVecLister(ctx, zctx, r, pool, commit)
}

func NewVecListerFromSnap(ctx context.Context, zctx *zed.Context, r *lake.Root, pool *lake.Pool, snap commits.View) *VecLister {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	l := &VecLister{
		ctx:       ctx,
		pool:      pool,
		snap:      snap,
		group:     &errgroup.Group{},
		marshaler: m,
	}
	return l
}

func (l *VecLister) Snapshot() commits.View {
	return l.snap
}

func (l *VecLister) Pull(done bool) (zbuf.Batch, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return nil, l.err
	}
	if l.vectors == nil {
		l.vectors = initVectorScan(l.snap)
	}
	for len(l.vectors) != 0 {
		o := l.vectors[0]
		l.vectors = l.vectors[1:]
		val, err := l.marshaler.Marshal(o)
		if err != nil {
			l.err = err
			return nil, err
		}
		// TODO Filter vectors by column metadata.
		return zbuf.NewArray([]zed.Value{*val}), nil
	}
	return nil, nil
}

func initVectorScan(snap commits.View) []ksuid.KSUID {
	objects := snap.SelectAllVectors()
	return objects
}
