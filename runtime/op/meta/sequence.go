package meta

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

// SequenceScanner implements an op that pulls metadata partitions to scan
// from its parent and for each partition, scans the object.
type SequenceScanner struct {
	parent      zbuf.Puller
	current     zbuf.Puller
	filter      zbuf.Filter
	pruner      expr.Evaluator
	octx        *op.Context
	pool        *lake.Pool
	progress    *zbuf.Progress
	snap        commits.View
	unmarshaler *zson.UnmarshalZNGContext
	done        bool
	err         error
	cmp         expr.CompareFn
}

func NewSequenceScanner(octx *op.Context, parent zbuf.Puller, pool *lake.Pool, snap commits.View, filter zbuf.Filter, pruner expr.Evaluator, progress *zbuf.Progress) *SequenceScanner {
	return &SequenceScanner{
		octx:        octx,
		parent:      parent,
		filter:      filter,
		pruner:      pruner,
		pool:        pool,
		progress:    progress,
		snap:        snap,
		unmarshaler: zson.NewZNGUnmarshaler(),
		cmp:         expr.NewValueCompareFn(order.Asc, true),
	}
}

func (s *SequenceScanner) Pull(done bool) (zbuf.Batch, error) {
	if s.done {
		return nil, s.err
	}
	if done {
		if s.current != nil {
			_, err := s.current.Pull(true)
			s.close(err)
			s.current = nil
		}
		return nil, s.err
	}
	for {
		if s.current == nil {
			if s.parent == nil { //XXX
				s.close(nil)
				return nil, nil
			}
			// Pull the next partition from the parent snapshot and
			// set up the next scanner to pull from.
			var part Partition
			ok, err := nextPartition(s.parent, &part, s.unmarshaler)
			if !ok || err != nil {
				s.close(err)
				return nil, err
			}
			s.current, err = newSortedPartitionScanner(s, part)
			if err != nil {
				s.close(err)
				return nil, err
			}
		}
		batch, err := s.current.Pull(false)
		if err != nil {
			s.close(err)
			return nil, err
		}
		if batch != nil {
			return batch, nil
		}
		s.current = nil
	}
}

func (s *SequenceScanner) close(err error) {
	s.err = err
	s.done = true
}

func nextPartition(puller zbuf.Puller, part *Partition, u *zson.UnmarshalZNGContext) (bool, error) {
	batch, err := puller.Pull(false)
	if batch == nil || err != nil {
		return false, err
	}
	vals := batch.Values()
	if len(vals) != 1 {
		// We currently support only one partition per batch.
		return false, errors.New("system error: SequenceScanner encountered multi-valued batch")
	}
	return true, u.Unmarshal(&vals[0], part)
}

func newSortedPartitionScanner(p *SequenceScanner, part Partition) (zbuf.Puller, error) {
	pullers := make([]zbuf.Puller, 0, len(part.Objects))
	pullersDone := func() {
		for _, puller := range pullers {
			puller.Pull(true)
		}
	}
	// Need to wrap the pruner with a pruner that also prunes the Min/Max of
	// the partition.
	pruner := newPartitionPruner(p.octx.Zctx, p.pruner, part, p.cmp)
	// Need to also wrap filter so it filters out partition Min/Max
	// func newPartitionFilter(zctx *zed.Context, span *extent.Generic, f zbuf.Filter, poolkey field.Path) *partitionFilter {
	filter := newPartitionFilter(p.octx.Zctx, p.cmp, part, p.filter, p.pool.Layout.Primary())
	var size int64
	for _, object := range part.Objects {
		ranges, err := data.LookupSeekRange(p.octx.Context, p.pool.Storage(), p.pool.DataPath, object, pruner)
		if err != nil {
			return nil, err
		}
		var osize int64
		for _, r := range ranges {
			size += r.Length
			osize += r.Length
		}
		// fmt.Println("object", object.ID, "ranges", len(ranges), "size", osize, "part.min/max", zson.String(part.Min), zson.String(part.Max))
		rc, err := object.NewReader(p.octx.Context, p.pool.Storage(), p.pool.DataPath, ranges)
		if err != nil {
			pullersDone()
			return nil, err
		}
		scanner, err := zngio.NewReader(p.octx.Zctx, rc).NewScanner(p.octx.Context, filter)
		if err != nil {
			pullersDone()
			rc.Close()
			return nil, err
		}
		pullers = append(pullers, &statScanner{
			scanner:  scanner,
			closer:   rc,
			progress: p.progress,
		})
	}
	// fmt.Println("partition.size", size, "min", zson.String(part.Min), "max", zson.String(part.Max))
	if len(pullers) == 1 {
		return pullers[0], nil
	}
	return merge.New(p.octx.Context, pullers, lake.ImportComparator(p.octx.Zctx, p.pool).Compare), nil
}

type partitionPruner struct {
	min, max expr.Evaluator
	part     Partition
	cmp      expr.CompareFn
	pruner   expr.Evaluator
}

func newPartitionPruner(zctx *zed.Context, pruner expr.Evaluator, part Partition, cmp expr.CompareFn) expr.Evaluator {
	return &partitionPruner{
		min:    expr.NewDottedExpr(zctx, field.New("min")),
		max:    expr.NewDottedExpr(zctx, field.New("max")),
		part:   part,
		cmp:    cmp,
		pruner: pruner,
	}
}

func (p *partitionPruner) Eval(ectx expr.Context, val *zed.Value) *zed.Value {
	if p.pruner != nil {
		result := p.pruner.Eval(ectx, val)
		if result.Type != zed.TypeBool || !zed.IsTrue(result.Bytes) {
			return result
		}
	}
	min := p.min.Eval(ectx, val).MissingAsNull()
	max := p.max.Eval(ectx, val).MissingAsNull()
	if p.part.Overlaps(p.cmp, min, max) {
		return zed.False
	}
	return zed.True
}

type partitionFilter struct {
	zbuf.Filter
	part Partition
	cmp  expr.CompareFn
	pk   expr.Evaluator
}

func newPartitionFilter(zctx *zed.Context, cmp expr.CompareFn, part Partition, f zbuf.Filter, poolkey field.Path) *partitionFilter {
	pk := expr.NewDottedExpr(zctx, poolkey)
	return &partitionFilter{
		Filter: f,
		part:   part,
		cmp:    cmp,
		pk:     pk,
	}
}

func (p *partitionFilter) AsEvaluator() (expr.Evaluator, error) {
	var e expr.Evaluator
	if p.Filter != nil {
		var err error
		if e, err = p.Filter.AsEvaluator(); err != nil {
			return nil, err
		}
	}
	return expr.EvaluatorFunc(func(ectx expr.Context, val *zed.Value) *zed.Value {
		pk := p.pk.Eval(ectx, val).MissingAsNull()
		if p.part.In(p.cmp, pk) {
			if e != nil {
				return e.Eval(ectx, val)
			}
			return zed.True
		}
		return zed.False
	}), nil
}

func (p *partitionFilter) AsBufferFilter() (*expr.BufferFilter, error) {
	if p.Filter == nil {
		return nil, nil
	}
	return p.Filter.AsBufferFilter()
}

// AsBufferFilter() (*expr.BufferFilter, error)

type statScanner struct {
	scanner  zbuf.Scanner
	closer   io.Closer
	err      error
	progress *zbuf.Progress
}

func (s *statScanner) Pull(done bool) (zbuf.Batch, error) {
	if s.scanner == nil {
		return nil, s.err
	}
	batch, err := s.scanner.Pull(done)
	if batch == nil || err != nil {
		s.progress.Add(s.scanner.Progress())
		if err2 := s.closer.Close(); err == nil {
			err = err2
		}
		s.err = err
		s.scanner = nil
	}
	return batch, err
}
