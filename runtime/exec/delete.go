package exec

import (
	"context"
	"io"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/runtime/meta"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/segmentio/ksuid"
	"golang.org/x/exp/slices"
)

type DeletePlanner struct {
	*Planner
	mu     sync.Mutex
	delete []ksuid.KSUID
}

func NewDeletePlanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, filter zbuf.Filter) (*DeletePlanner, error) {
	p, err := NewPlannerByID(ctx, zctx, r, poolID, commit, filter)
	if err != nil {
		return nil, err
	}
	return &DeletePlanner{Planner: p.(*Planner)}, nil
}

func (p *DeletePlanner) PullWork() (zbuf.Puller, error) {
	p.once.Do(func() {
		p.run()
	})
	select {
	case part := <-p.ch:
		if part.Objects == nil {
			return nil, p.group.Wait()
		}
		return newDeletePartitionScanner(p, part)
	case <-p.ctx.Done():
		return nil, p.group.Wait()
	}
}

func (p *DeletePlanner) deleteObject(id ksuid.KSUID) {
	p.mu.Lock()
	p.delete = append(p.delete, id)
	p.mu.Unlock()
}

func (p *DeletePlanner) DeleteObjects() []ksuid.KSUID {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.delete)
}

func newDeletePartitionScanner(p *DeletePlanner, part meta.Partition) (zbuf.Puller, error) {
	pullers := make([]zbuf.Puller, 0, len(part.Objects))
	pullersDone := func() {
		for _, puller := range pullers {
			puller.Pull(true)
		}
	}
	for _, o := range part.Objects {
		rg, err := p.objectRange(o)
		if err != nil {
			return nil, err
		}
		rc, err := o.NewReader(p.ctx, p.pool.Storage(), p.pool.DataPath, rg)
		if err != nil {
			pullersDone()
			return nil, err
		}
		scanner, err := zngio.NewReader(p.zctx, rc).NewScanner(p.ctx, p.filter)
		if err != nil {
			pullersDone()
			rc.Close()
			return nil, err
		}
		pullers = append(pullers, &deleteScanner{
			object:  o,
			scanner: scanner,
			closer:  rc,
			planner: p,
		})
	}
	if len(pullers) == 1 {
		return pullers[0], nil
	}
	return merge.New(p.ctx, pullers, lake.ImportComparator(p.zctx, p.pool).Compare), nil
}

type deleteScanner struct {
	object  *data.Object
	once    sync.Once
	batches []zbuf.Batch
	planner *DeletePlanner
	scanner zbuf.Scanner
	closer  io.Closer
	err     error
}

func (s *deleteScanner) Pull(done bool) (zbuf.Batch, error) {
	s.once.Do(func() {
		for {
			batch, err := s.scanner.Pull(done)
			if batch == nil || err != nil {
				s.planner.progress.Add(s.scanner.Progress())
				if err2 := s.closer.Close(); err == nil {
					err = err2
				}
				s.err = err
				var count uint64
				for _, b := range s.batches {
					count += uint64(len(b.Values()))
				}
				if count == s.object.Count {
					// Only return batches from objects where values have been
					// deleted.
					s.batches = nil
				} else {
					s.planner.deleteObject(s.object.ID)
				}
				return
			}
			s.batches = append(s.batches, batch)
		}
	})
	if len(s.batches) == 0 || s.err != nil {
		return nil, s.err
	}
	batch := s.batches[0]
	s.batches = s.batches[1:]
	return batch, nil
}
