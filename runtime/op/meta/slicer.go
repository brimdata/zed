package meta

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

// Slicer implements an op that pulls data objects and organizes
// them into overlapping object Slices forming a sequence of
// non-overlapping Partitions.
type Slicer struct {
	ctx         context.Context
	parent      zbuf.Puller
	marshaler   *zson.MarshalZNGContext
	unmarshaler *zson.UnmarshalZNGContext
	objects     []*data.Object
	cmp         expr.CompareFn
	min         *zed.Value
	max         *zed.Value
	mu          sync.Mutex
	pool        *lake.Pool
	breaker     *breaker
}

func NewSlicer(ctx context.Context, parent zbuf.Puller, zctx *zed.Context, pool *lake.Pool) *Slicer {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return &Slicer{
		ctx:         ctx,
		parent:      parent,
		marshaler:   m,
		unmarshaler: zson.NewZNGUnmarshaler(),
		//XXX check nullsmax is consistent for both dirs in lake ops
		cmp:  expr.NewValueCompareFn(order.Asc, true),
		pool: pool,
	}
}

func (s *Slicer) Snapshot() commits.View {
	//XXX
	return s.parent.(*Lister).Snapshot()
}

func (s *Slicer) Pull(done bool) (zbuf.Batch, error) {
	//XXX for now we use a mutex because multiple downstream trunks can call
	// Pull concurrently here.  We should change this to use a fork.  But for now,
	// this does not seem like a performance critical issue because the bottleneck
	// will be each trunk and the lister parent should run fast in comparison.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.breaker != nil {
		part, err := s.breaker.next()
		if part != nil && err == nil {
			return partitionBatch(s.marshaler, part)
		}
		s.breaker = nil
	}
	for {
		batch, err := s.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return s.nextPartition()
		}
		vals := batch.Values()
		if len(vals) != 1 {
			// We currently support only one object per batch.
			return nil, errors.New("system error: Slicer encountered multi-valued batch")
		}
		var object data.Object
		if err := s.unmarshaler.Unmarshal(&vals[0], &object); err != nil {
			return nil, err
		}
		if batch, err := s.stash(&object); batch != nil || err != nil {
			return batch, err
		}
	}
}

// nextPartition takes collected up slices and forms a partition returning
// a batch containing a single value comprising the serialized partition.
func (s *Slicer) nextPartition() (zbuf.Batch, error) {
	if len(s.objects) == 0 {
		return nil, nil
	}
	//XXX let's keep this as we go!... need to reorder stuff in stash() to make this work
	min := &s.objects[0].Min
	max := &s.objects[0].Max
	size := s.objects[0].Size
	for _, o := range s.objects[1:] {
		if s.cmp(&o.Min, min) < 0 {
			min = &o.Min
		}
		if s.cmp(&o.Max, max) > 0 {
			max = &o.Max
		}
		size += o.Size
	}
	var part *Partition
	if size > s.pool.Threshold*2 {
		// If size is greater than 2x the pool threshold we need to break things up.
		// The only way to break to find suitable partitions is by querying
		// the seek index for all objects in the main partition and find out where
		// it make sense to break the objects up. Fortunately the seek indexes
		// should get cached so the io cost of fetching them will only be incurred
		// once.
		seeks, err := fetchSeekIndexes(s.ctx, s.pool, s.objects)
		if err != nil {
			return nil, err
		}
		s.breaker = newBreaker(s, s.objects, seeks)
		part, err = s.breaker.next()
		if err != nil {
			return nil, err
		}
	} else {
		part = &Partition{
			Min:     min,
			Max:     max,
			Objects: s.objects,
		}
	}
	s.objects = s.objects[:0]
	return partitionBatch(s.marshaler, part)
}

func partitionBatch(marshaler *zson.MarshalZNGContext, part *Partition) (zbuf.Batch, error) {
	val, err := marshaler.Marshal(part)
	if err != nil {
		return nil, err
	}
	return zbuf.NewArray([]zed.Value{*val}), nil
}

func (s *Slicer) stash(o *data.Object) (zbuf.Batch, error) {
	var batch zbuf.Batch
	if len(s.objects) > 0 {
		// We collect all the subsequent objects that overlap with any object in the
		// accumulated set so far.  Since first times are non-decreasing this is
		// guaranteed to generate partitions that are non-decreasing and non-overlapping.
		if s.cmp(&o.Max, s.min) < 0 || s.cmp(&o.Min, s.max) > 0 {
			var err error
			batch, err = s.nextPartition()
			if err != nil {
				return nil, err
			}
			s.min = nil
			s.max = nil
		}
	}
	s.objects = append(s.objects, o)
	if s.min == nil {
		s.min = o.Min.Copy()
		s.max = o.Max.Copy()
	} else {
		if s.cmp(s.min, &o.Min) > 0 {
			s.min = o.Min.Copy()
		}
		if s.cmp(s.max, &o.Max) < 0 {
			s.max = o.Max.Copy()
		}
	}
	return batch, nil
}

type item struct {
	index int
	val   *zed.Value
}

type breaker struct {
	slicer  *Slicer
	seeks   [][]seekindex.Entry
	lastMax *zed.Value
	objects []*data.Object
	order   []int
}

func newBreaker(slicer *Slicer, objects []*data.Object, seeks [][]seekindex.Entry) *breaker {
	type item struct {
		objIndex int
		val      *zed.Value
	}
	var items []item
	for i := range seeks {
		for j := range seeks[i] {
			items = append(items, item{i, seeks[i][j].Max})
		}
	}
	slices.SortFunc(items, func(a, b item) bool {
		return slicer.cmp(a.val, b.val) < 0
	})
	order := make([]int, len(items))
	for i, it := range items {
		order[i] = it.objIndex
	}
	return &breaker{
		slicer:  slicer,
		objects: objects,
		seeks:   slices.Clone(seeks),
		order:   order,
	}
}

func (b *breaker) next() (*Partition, error) {
	var size int64
	var span *extent.Generic
	for len(b.order) > 0 && size < b.slicer.pool.Threshold {
		i := b.order[0]
		seek := b.seeks[i][0]
		b.order, b.seeks[i] = b.order[1:], b.seeks[i][1:]
		if span == nil {
			span = extent.NewGeneric(*b.nextMin(), *seek.Max.Copy(), b.slicer.cmp)
		} else {
			span.Extend(seek.Max)
		}
		size += int64(seek.Length)
	}
	if span == nil {
		return nil, nil
	}
	// Add objects overlapping objects to the map.
	var objects []*data.Object
	for _, o := range b.objects {
		if span.Overlaps(&o.Min, &o.Max) {
			objects = append(objects, o)
		}
	}
	// fmt.Fprintln(os.Stderr, "newpartition", "objects", len(objects), "size", size, "first", zson.String(span.First()), "last", zson.String(span.Last()))
	p := &Partition{
		Min:     span.First(),
		MinOpen: b.lastMax != nil,
		Max:     span.Last(),
		Objects: objects,
	}
	b.lastMax = p.Max.Copy()
	return p, nil
}

func (b *breaker) nextMin() *zed.Value {
	if b.lastMax != nil {
		return b.lastMax.Copy()
	}
	min := b.objects[0].Min.Copy()
	for _, o := range b.objects[1:] {
		if b.slicer.cmp(&o.Min, min) < 0 {
			min = o.Min.Copy()
		}
	}
	return min
}

// func (b *breaker) nextMin() (*zed.Value, error) {
// 	// Calculate the next partition's Minimum value which will be the smallest
// 	// value greater than the max of the previous batch. If this is the first
// 	// Partition then min will be the minimum value of all objects.
// 	if b.lastMax == nil {
// 		min := b.objects[0].Min.Copy()
// 		for _, o := range b.objects[1:] {
// 			if b.slicer.cmp(&o.Min, min) < 0 {
// 				min = o.Min.Copy()
// 			}
// 		}
// 		return min, nil
// 	}
// 	// Cracks knuckles. Need to find the next smallest value greater than
// 	// lastMax. This could either be the minimum of one of the remaining seeks
// 	// but if a seek entry is bisected by lastMax we will have to
// 	var scanObjs []*data.Object
// 	var scanSeeks []seekindex.Entry
// 	var min *zed.Value
// 	for i, seek := range b.seeks {
// 		entry := seek[0]
// 		if b.slicer.cmp(entry.Min, b.lastMax) > 0 {
// 			if min == nil || b.slicer.cmp(entry.Min, min) < 0 {
// 				min = entry.Min
// 			}
// 			continue
// 		}
// 		scanObjs, scanSeeks = append(scanObjs, b.objects[i]), append(scanSeeks, entry)
// 	}
// 	if len(scanObjs) > 0 {
// 		var mu sync.Mutex
// 		group, ctx := errgroup.WithContext(b.slicer.ctx)
// 		for i, o := range scanObjs {
// 			o, entry := o, scanSeeks[i]
// 			group.Go(func() error {
// 				val, err := b.findGreaterThanLastMax(ctx, o, entry, b.lastMax.Copy())
// 				if err != nil {
// 					return err
// 				}
// 				mu.Lock()
// 				if min == nil || b.slicer.cmp(val, min) < 0 {
// 					min = val
// 				}
// 				mu.Unlock()
// 				return nil
// 			})
// 		}
// 		if err := group.Wait(); err != nil {
// 			return nil, err
// 		}
// 	}
// 	return min, nil
// }
//
// func (b *breaker) findGreaterThanLastMax(ctx context.Context, o *data.Object, entry seekindex.Entry, max *zed.Value) (*zed.Value, error) {
// 	// func (o *Object) NewReader(ctx context.Context, engine storage.Engine, path *storage.URI, ranges []seekindex.Range) (*Reader, error) {
// 	pool := b.slicer.pool
// 	rg := []seekindex.Range{{int64(entry.Offset), int64(entry.Length)}}
// 	r, err := o.NewReader(ctx, pool.Storage(), pool.DataPath, rg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer r.Close()
// 	zr := zngio.NewReader
// 	for {
// 		val, err := r.Read()
// 		if val == nil || err != nil {
// 			return nil, err
// 		}
// 	}
// 	return nil, nil
// }

type seek []seekindex.Entry

func fetchSeekIndexes(ctx context.Context, pool *lake.Pool, objects []*data.Object) ([][]seekindex.Entry, error) {
	group, ctx := errgroup.WithContext(ctx)
	seeks := make([][]seekindex.Entry, len(objects))
	engine := pool.Storage()
	var mu sync.Mutex
	for i, o := range objects {
		i, o := i, o
		group.Go(func() error {
			seek, err := data.FetchSeekIndex(ctx, engine, pool.DataPath, o)
			if err != nil {
				return err
			}
			mu.Lock()
			seeks[i] = seek
			mu.Unlock()
			return nil
		})
	}
	err := group.Wait()
	return seeks, err
}

// A Partition is a logical view of the records within a pool-key span, stored
// in one or more data objects.  This provides a way to return the list of
// objects that should be scanned along with a span to limit the scan
// to only the span involved.
type Partition struct {
	Min     *zed.Value     `zed:"min"`
	MinOpen bool           `zed:"min_open"`
	Max     *zed.Value     `zed:"max"`
	MaxOpen bool           `zed:"max_open"`
	Objects []*data.Object `zed:"objects"`
}

func (p Partition) IsZero() bool {
	return p.Objects == nil
}

func (p Partition) FormatRangeOf(index int) string {
	o := p.Objects[index]
	return fmt.Sprintf("[%s-%s,%s-%s]", zson.String(p.Min), zson.String(p.Max), zson.String(o.Min), zson.String(o.Max))
}

func (p Partition) FormatRange() string {
	return fmt.Sprintf("[%s-%s]", zson.String(p.Min), zson.String(p.Max))
}

// In returns true if val is located within the partition.
func (p *Partition) In(cmp expr.CompareFn, val *zed.Value) bool {
	if i := cmp(val, p.Min); i < 0 || (!p.MinOpen && i == 0) {
		return false
	}
	if i := cmp(val, p.Max); i > 0 || (!p.MaxOpen && i == 0) {
		return false
	}
	return true
}

func (p *Partition) Overlaps(cmp expr.CompareFn, min, max *zed.Value) bool {
	if i := cmp(min, p.Min); i > 0 || (!p.MinOpen && i == 0) {
		i = cmp(min, p.Max)
		return i < 0 || (!p.MaxOpen && i == 0)
	}
	i := cmp(max, p.Min)
	return i > 0 || (!p.MinOpen && i == 0)
}
