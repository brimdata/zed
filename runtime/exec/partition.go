package exec

import (
	"bytes"
	"context"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/meta"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

// PartitionObjects takes a sorted set of data objects with possibly overlapping
// key ranges and returns an ordered list of Ranges such that none of the
// Ranges overlap with one another.  This is the straightforward computational
// geometry problem of merging overlapping intervals,
// e.g., https://www.geeksforgeeks.org/merging-intervals/
//
// XXX this algorithm doesn't quite do what we want because it continues
// to merge *anything* that overlaps.  It's easy to fix though.
// Issue #2538
func PartitionObjects(objects []*data.Object, o order.Which) []meta.Partition {
	if len(objects) == 0 {
		return nil
	}
	cmp := extent.CompareFunc(o)
	spans := sortedObjectSpans(objects, cmp)
	var s stack
	s.pushObjectSpan(spans[0], cmp)
	for _, span := range spans[1:] {
		tos := s.tos()
		if span.Before(tos.Last()) {
			s.pushObjectSpan(span, cmp)
		} else {
			tos.Objects = append(tos.Objects, span.object)
			tos.Extend(span.Last())
		}
	}
	// On exit, the ranges in the stack are properly sorted so
	// we just return the stack as a []Range.
	return s
}

type stack []meta.Partition

func (s *stack) pushObjectSpan(span objectSpan, cmp expr.CompareFn) {
	s.push(meta.Partition{
		Span:    span.Span,
		Objects: []*data.Object{span.object},
	})
}

func (s *stack) push(p meta.Partition) {
	*s = append(*s, p)
}

func (s *stack) pop() meta.Partition {
	n := len(*s)
	p := (*s)[n-1]
	*s = (*s)[:n-1]
	return p
}

func (s *stack) tos() *meta.Partition {
	return &(*s)[len(*s)-1]
}

type objectSpan struct {
	extent.Span
	object *data.Object
}

func sortedObjectSpans(objects []*data.Object, cmp expr.CompareFn) []objectSpan {
	spans := make([]objectSpan, 0, len(objects))
	for _, o := range objects {
		spans = append(spans, objectSpan{
			Span:   extent.NewGeneric(o.First, o.Last, cmp),
			object: o,
		})
	}
	sort.Slice(spans, func(i, j int) bool {
		return objectSpanLess(spans[i], spans[j])
	})
	return spans
}

func objectSpanLess(a, b objectSpan) bool {
	if b.Before(a.First()) {
		return true
	}
	if !bytes.Equal(a.First().Bytes, b.First().Bytes) {
		return false
	}
	if bytes.Equal(a.Last().Bytes, b.Last().Bytes) {
		if a.object.Count != b.object.Count {
			return a.object.Count < b.object.Count
		}
		return ksuid.Compare(a.object.ID, b.object.ID) < 0
	}
	return a.After(b.Last())
}

func sortObjects(o order.Which, objects []*data.Object) {
	for k, span := range sortedObjectSpans(objects, extent.CompareFunc(o)) {
		objects[k] = span.object
	}
}

func partitionReader(ctx context.Context, zctx *zed.Context, layout order.Layout, snap commits.View, filter zbuf.Filter) (zio.Reader, error) {
	ch := make(chan meta.Partition)
	ctx, cancel := context.WithCancel(ctx)
	var scanErr error
	go func() {
		scanErr = ScanPartitions(ctx, snap, layout, filter, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return readerFunc(func() (*zed.Value, error) {
		select {
		case p := <-ch:
			if p.Objects == nil {
				cancel()
				return nil, scanErr
			}
			rec, err := m.Marshal(p)
			if err != nil {
				cancel()
				return nil, err
			}
			return rec, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}), nil
}

func objectReader(ctx context.Context, zctx *zed.Context, snap commits.View, order order.Which) (zio.Reader, error) {
	ch := make(chan data.Object)
	ctx, cancel := context.WithCancel(ctx)
	var scanErr error
	go func() {
		scanErr = Scan(ctx, snap, order, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return readerFunc(func() (*zed.Value, error) {
		select {
		case p := <-ch:
			if p.ID == ksuid.Nil {
				cancel()
				return nil, scanErr
			}
			rec, err := m.Marshal(p)
			if err != nil {
				cancel()
				return nil, err
			}
			return rec, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}), nil
}

func indexObjectReader(ctx context.Context, zctx *zed.Context, snap commits.View, order order.Which) (zio.Reader, error) {
	ch := make(chan *index.Object)
	ctx, cancel := context.WithCancel(ctx)
	var scanErr error
	go func() {
		scanErr = ScanIndexes(ctx, snap, order, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return readerFunc(func() (*zed.Value, error) {
		select {
		case p := <-ch:
			if p == nil {
				cancel()
				return nil, scanErr
			}
			rec, err := m.Marshal(*p)
			if err != nil {
				cancel()
				return nil, err
			}
			return rec, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}), nil
}

type readerFunc func() (*zed.Value, error)

func (r readerFunc) Read() (*zed.Value, error) { return r() }
