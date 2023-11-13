package meta

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

// VecSequenceScanner pulls vector ids from its parent and, for each id, scans the vector.
type VecSequenceScanner struct {
	parent      zbuf.Puller
	scanner     zbuf.Puller
	octx        *op.Context
	pool        *lake.Pool
	progress    *zbuf.Progress
	unmarshaler *zson.UnmarshalZNGContext
	demand      demand.Demand
	done        bool
	err         error
}

func NewVecSequenceScanner(octx *op.Context, parent zbuf.Puller, pool *lake.Pool, progress *zbuf.Progress, demandOut demand.Demand) *VecSequenceScanner {
	return &VecSequenceScanner{
		octx:        octx,
		parent:      parent,
		pool:        pool,
		progress:    progress,
		unmarshaler: zson.NewZNGUnmarshaler(),
		demand:      demandOut,
	}
}

func (s *VecSequenceScanner) Pull(done bool) (zbuf.Batch, error) {
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
				err := errors.New("system error: VecSequenceScanner encountered multi-valued batch")
				s.close(err)
				return nil, err
			}
			s.scanner, _, err = newVecSequenceScanner(s.octx.Context, s.octx.Zctx, s.pool, s.unmarshaler, s.progress, &vals[0], s.demand)
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

func (s *VecSequenceScanner) close(err error) {
	s.err = err
	s.done = true
}

func newVecSequenceScanner(ctx context.Context, zctx *zed.Context, pool *lake.Pool, u *zson.UnmarshalZNGContext, progress *zbuf.Progress, val *zed.Value, demandOut demand.Demand) (zbuf.Puller, ksuid.KSUID, error) {
	named, ok := val.Type.(*zed.TypeNamed)
	if !ok {
		return nil, ksuid.KSUID{}, errors.New("system error: VecSequenceScanner encountered unnamed object")
	}
	if named.Name != "ksuid.KSUID" {
		return nil, ksuid.KSUID{}, fmt.Errorf("system error: VecSequenceScanner encountered an object named %s", named.Name)
	}
	var id ksuid.KSUID
	if err := u.Unmarshal(val, &id); err != nil {
		return nil, ksuid.KSUID{}, err
	}
	storage := pool.Storage()
	uri := data.VectorURI(pool.DataPath, id)
	size, err := storage.Size(ctx, uri)
	if err != nil {
		return nil, ksuid.KSUID{}, err
	}
	ioReader, err := storage.Get(ctx, uri)
	if err != nil {
		return nil, ksuid.KSUID{}, err
	}
	object, err := vng.NewObject(zctx, ioReader, size)
	if err != nil {
		return nil, ksuid.KSUID{}, err
	}
	vectorReader := vector.NewReader(object, demandOut)
	scanner, err := zbuf.NewScanner(ctx, vectorReader, nil)
	if err != nil {
		return nil, ksuid.KSUID{}, err
	}
	return &vecStatScanner{
		scanner:  scanner,
		progress: progress,
	}, id, nil
}

type vecStatScanner struct {
	scanner  zbuf.Scanner
	err      error
	progress *zbuf.Progress
}

func (s *vecStatScanner) Pull(done bool) (zbuf.Batch, error) {
	if s.scanner == nil {
		return nil, s.err
	}
	batch, err := s.scanner.Pull(done)
	if batch == nil || err != nil {
		s.progress.Add(s.scanner.Progress())
		s.err = err
		s.scanner = nil
	}
	return batch, err
}
