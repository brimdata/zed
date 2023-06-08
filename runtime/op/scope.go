package op

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
	"golang.org/x/exp/slices"
)

type enterScope struct {
	parent zbuf.Puller
	vars   []zed.Value
}

func NewEnterScope(parent zbuf.Puller, vars []zed.Value) zbuf.Puller {
	if len(vars) == 0 {
		return parent
	}
	return &enterScope{parent, vars}
}

func (s *enterScope) Pull(done bool) (zbuf.Batch, error) {
	batch, err := s.parent.Pull(done)
	if batch != nil {
		vars := append(slices.Clone(batch.Vars()), s.vars...)
		batch = &scopedBatch{batch, vars}
	}
	return batch, err
}

type exitScope struct {
	parent zbuf.Puller
	nvars  int
}

func NewExitScope(parent zbuf.Puller, nvars int) zbuf.Puller {
	if nvars == 0 {
		return parent
	}
	return &exitScope{parent, nvars}
}

func (s *exitScope) Pull(done bool) (zbuf.Batch, error) {
	batch, err := s.parent.Pull(done)
	if batch != nil {
		vars := batch.Vars()
		vars = vars[:len(vars)-s.nvars]
		batch = &scopedBatch{batch, vars}
	}
	return batch, err
}

var _ zbuf.Batch = (*scopedBatch)(nil)

type scopedBatch struct {
	zbuf.Batch
	vars []zed.Value
}

func (s *scopedBatch) Vars() []zed.Value {
	return s.vars
}
