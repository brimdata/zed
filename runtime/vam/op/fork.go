package op

import (
	"context"
	"sync"

	"github.com/brimdata/super/vector"
)

type Fork struct {
	ctx    context.Context
	parent vector.Puller

	branches []*forkBranch
	nblocked int
	once     sync.Once
}

func NewFork(ctx context.Context, parent vector.Puller) *Fork {
	return &Fork{
		ctx:    ctx,
		parent: parent,
	}
}

func (f *Fork) AddExit() vector.Puller {
	branch := &forkBranch{f, make(chan result), make(chan struct{}), false}
	f.branches = append(f.branches, branch)
	return branch
}

func (f *Fork) run() {
	for {
		if f.nblocked == len(f.branches) {
			// Send done upstream.
			if _, err := f.parent.Pull(true); err != nil {
				for _, b := range f.branches {
					select {
					case b.resultCh <- result{nil, err}:
					case <-f.ctx.Done():
					}
				}
				return
			}
			f.unblockBranches()
		}
		vec, err := f.parent.Pull(false)
		for _, b := range f.branches {
			if b.blocked {
				continue
			}
			select {
			case b.resultCh <- result{vec, err}:
			case <-b.doneCh:
				b.blocked = true
				f.nblocked++
			case <-f.ctx.Done():
				return
			}
		}
		if vec == nil && err == nil {
			// EOS unblocks all branches.
			f.unblockBranches()
		}
	}
}

func (f *Fork) unblockBranches() {
	for _, b := range f.branches {
		b.blocked = false
	}
	f.nblocked = 0
}

type forkBranch struct {
	fork     *Fork
	resultCh chan result
	doneCh   chan struct{}
	blocked  bool
}

func (f *forkBranch) Pull(done bool) (vector.Any, error) {
	f.fork.once.Do(func() { go f.fork.run() })
	if done {
		select {
		case f.doneCh <- struct{}{}:
			return nil, nil
		case <-f.fork.ctx.Done():
			return nil, f.fork.ctx.Err()
		}
	}
	select {
	case r := <-f.resultCh:
		return r.vector, r.err
	case <-f.fork.ctx.Done():
		return nil, f.fork.ctx.Err()
	}
}
