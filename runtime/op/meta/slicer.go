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
	partitions  []*Partition
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
	if len(s.partitions) > 0 {
		p := s.partitions[0]
		s.partitions = s.partitions[1:]
		return batchifyPart(s.marshaler, p)
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
		var err error
		if s.partitions, err = s.breakup(); err != nil {
			return nil, err
		}
		part = s.partitions[0]
		s.partitions = s.partitions[1:]
	} else {
		part = &Partition{
			Min:     min,
			Max:     max,
			Objects: s.objects,
		}
	}
	s.objects = s.objects[:0]
	return batchifyPart(s.marshaler, part)
}

func batchifyPart(marshaler *zson.MarshalZNGContext, part *Partition) (zbuf.Batch, error) {
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

func (s *Slicer) breakup() ([]*Partition, error) {
	seeks, err := fetchSeeks(s.ctx, s.cmp, s.pool, s.objects)
	if err != nil {
		return nil, err
	}
	var partitions []*Partition
	var size int64
	span := extent.NewGeneric(*s.min, *s.min, s.cmp)
	for _, seek := range seeks {
		if size >= s.pool.Threshold && s.cmp(seek.Max, span.Last()) != 0 {
			// Every partition except for the first partition should be open,
			// closed.
			minOpen := len(partitions) > 0
			p := partitionFromSpan(s.cmp, span, s.objects, minOpen)
			partitions = append(partitions, p)
			// Start new span at end of current partition.
			span = extent.NewGeneric(*p.Max.Copy(), *p.Max.Copy(), s.cmp)
			size = 0
		}
		span.Extend(seek.Max)
		size += int64(seek.Length)
	}
	if p := partitionFromSpan(s.cmp, span, s.objects, len(partitions) > 0); !p.IsZero() {
		partitions = append(partitions, p)
	}
	if s.pool.SortKey.Order == order.Desc {
		// Reverse partition order for descending.
		for i, j := 0, len(partitions)-1; i < j; i, j = i+1, j-1 {
			partitions[i], partitions[j] = partitions[j], partitions[i]
		}
	}
	return partitions, nil
}

func partitionFromSpan(cmp expr.CompareFn, span *extent.Generic, objects []*data.Object, minOpen bool) *Partition {
	p := &Partition{
		Min:     span.First().Copy(),
		MinOpen: minOpen,
		Max:     span.Last().Copy(),
	}
	// Add objects overlapping objects to the map.
	for _, o := range objects {
		if p.Overlaps(cmp, &o.Min, &o.Max) {
			p.Objects = append(p.Objects, o)
		}
	}
	return p
}

func fetchSeeks(ctx context.Context, cmp expr.CompareFn, pool *lake.Pool, objects []*data.Object) ([]seekindex.Entry, error) {
	group, ctx := errgroup.WithContext(ctx)
	var seeks []seekindex.Entry
	engine := pool.Storage()
	var mu sync.Mutex
	for _, o := range objects {
		o := o
		group.Go(func() error {
			s, err := data.FetchSeekIndex(ctx, engine, pool.DataPath, o)
			if err != nil {
				return err
			}
			mu.Lock()
			seeks = append(seeks, s...)
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(seeks, func(a, b seekindex.Entry) bool {
		return cmp(a.Max, b.Max) < 0
	})
	return seeks, nil
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
	if i := cmp(val, p.Min); i < 0 || (p.MinOpen && i == 0) {
		return false
	}
	if i := cmp(val, p.Max); i > 0 || (p.MaxOpen && i == 0) {
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
