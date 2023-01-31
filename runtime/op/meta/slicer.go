package meta

import (
	"errors"
	"fmt"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

// Slicer implements an op that pulls data objects and organizes
// them into overlapping object Slices forming a sequence of
// non-overlapping Partitions.
type Slicer struct {
	parent      zbuf.Puller
	marshaler   *zson.MarshalZNGContext
	unmarshaler *zson.UnmarshalZNGContext
	objects     []*data.Object
	cmp         expr.CompareFn
	last        *zed.Value
	mu          sync.Mutex
}

func NewSlicer(parent zbuf.Puller, zctx *zed.Context, o order.Which) *Slicer {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return &Slicer{
		parent:      parent,
		marshaler:   m,
		unmarshaler: zson.NewZNGUnmarshaler(),
		//XXX check nullsmax is consistent for both dirs in lake ops
		cmp: expr.NewValueCompareFn(o, o == order.Asc),
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
	first := &s.objects[0].First
	last := &s.objects[0].Last
	for _, o := range s.objects[1:] {
		if s.cmp(&o.First, first) < 0 {
			first = &o.First
		}
		if s.cmp(&o.Last, last) > 0 {
			last = &o.Last
		}
	}
	val, err := s.marshaler.Marshal(&Partition{
		First:   first,
		Last:    last,
		Objects: s.objects,
	})
	s.objects = s.objects[:0]
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
		if s.cmp(&o.First, s.last) >= 0 {
			var err error
			batch, err = s.nextPartition()
			if err != nil {
				return nil, err
			}
			s.last = nil
		}
	}
	s.objects = append(s.objects, o)
	if s.last == nil || s.cmp(s.last, &o.Last) < 0 {
		s.last = &o.Last
	}
	return batch, nil
}

// A Partition is a logical view of the records within a pool-key span, stored
// in one or more data objects.  This provides a way to return the list of
// objects that should be scanned along with a span to limit the scan
// to only the span involved.
type Partition struct {
	First   *zed.Value
	Last    *zed.Value
	Objects []*data.Object
}

func (p Partition) IsZero() bool {
	return p.Objects == nil
}

func (p Partition) FormatRangeOf(index int) string {
	o := p.Objects[index]
	return fmt.Sprintf("[%s-%s,%s-%s]", zson.String(p.First), zson.String(p.Last), zson.String(o.First), zson.String(o.Last))
}

func (p Partition) FormatRange() string {
	return fmt.Sprintf("[%s-%s]", zson.String(p.First), zson.String(p.Last))
}
