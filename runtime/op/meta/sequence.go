package meta

import (
	"context"
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/data"
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
	scanner     zbuf.Puller
	filter      zbuf.Filter
	pruner      expr.Evaluator
	octx        *op.Context
	pool        *lake.Pool
	progress    *zbuf.Progress
	unmarshaler *zson.UnmarshalZNGContext
	done        bool
	err         error
}

func NewSequenceScanner(octx *op.Context, parent zbuf.Puller, pool *lake.Pool, filter zbuf.Filter, pruner expr.Evaluator, progress *zbuf.Progress) *SequenceScanner {
	return &SequenceScanner{
		octx:        octx,
		parent:      parent,
		filter:      filter,
		pruner:      pruner,
		pool:        pool,
		progress:    progress,
		unmarshaler: zson.NewZNGUnmarshaler(),
	}
}

func (s *SequenceScanner) Pull(done bool) (zbuf.Batch, error) {
	if s.done {
		return nil, s.err
	}
	if done {
		if s.scanner != nil {
			_, err := s.scanner.Pull(true)
			s.close(err)
			s.scanner = nil
		}
		return nil, s.err
	}
	for {
		if s.scanner == nil {
			if s.parent == nil { //XXX
				s.close(nil)
				return nil, nil
			}
			batch, err := s.parent.Pull(false)
			if batch == nil || err != nil {
				s.close(err)
				return nil, err
			}
			vals := batch.Values()
			if len(vals) != 1 {
				// We currently support only one partition per batch.
				err := errors.New("system error: SequenceScanner encountered multi-valued batch")
				s.close(err)
				return nil, err
			}
			s.scanner, _, err = newScanner(s.octx.Context, s.octx.Zctx, s.pool, s.unmarshaler, s.pruner, s.filter, s.progress, &vals[0])
			if err != nil {
				s.close(err)
				return nil, err
			}
		}
		batch, err := s.scanner.Pull(false)
		if err != nil {
			s.close(err)
			return nil, err
		}
		if batch != nil {
			return batch, nil
		}
		s.scanner = nil
	}
}

func (s *SequenceScanner) close(err error) {
	s.err = err
	s.done = true
}

func newScanner(ctx context.Context, zctx *zed.Context, pool *lake.Pool, u *zson.UnmarshalZNGContext, pruner expr.Evaluator, filter zbuf.Filter, progress *zbuf.Progress, val *zed.Value) (zbuf.Puller, *data.Object, error) {
	named, ok := val.Type.(*zed.TypeNamed)
	if !ok {
		return nil, nil, errors.New("system error: SequenceScanner encountered unnamed object")
	}
	var objects []*data.Object
	if named.Name == "data.Object" {
		var object data.Object
		if err := u.Unmarshal(val, &object); err != nil {
			return nil, nil, err
		}
		objects = []*data.Object{&object}
	} else {
		var part Partition
		if err := u.Unmarshal(val, &part); err != nil {
			return nil, nil, err
		}
		objects = part.Objects
	}
	scanner, err := newObjectsScanner(ctx, zctx, pool, objects, pruner, filter, progress)
	return scanner, objects[0], err
}

func newObjectsScanner(ctx context.Context, zctx *zed.Context, pool *lake.Pool, objects []*data.Object, pruner expr.Evaluator, filter zbuf.Filter, progress *zbuf.Progress) (zbuf.Puller, error) {
	pullers := make([]zbuf.Puller, 0, len(objects))
	pullersDone := func() {
		for _, puller := range pullers {
			puller.Pull(true)
		}
	}
	for _, object := range objects {
		s, err := newObjectScanner(ctx, zctx, pool, object, filter, pruner, progress)
		if err != nil {
			pullersDone()
			return nil, err
		}
		pullers = append(pullers, s)
	}
	if len(pullers) == 1 {
		return pullers[0], nil
	}
	return merge.New(ctx, pullers, lake.ImportComparator(zctx, pool).Compare), nil
}

func newObjectScanner(ctx context.Context, zctx *zed.Context, pool *lake.Pool, object *data.Object, filter zbuf.Filter, pruner expr.Evaluator, progress *zbuf.Progress) (zbuf.Puller, error) {
	ranges, err := data.LookupSeekRange(ctx, pool.Storage(), pool.DataPath, object, pruner)
	if err != nil {
		return nil, err
	}
	rc, err := object.NewReader(ctx, pool.Storage(), pool.DataPath, ranges)
	if err != nil {
		return nil, err
	}
	scanner, err := zngio.NewReader(zctx, rc).NewScanner(ctx, filter)
	if err != nil {
		rc.Close()
		return nil, err
	}
	return &statScanner{
		scanner:  scanner,
		closer:   rc,
		progress: progress,
	}, nil
}

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
