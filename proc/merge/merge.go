package merge

import (
	"context"

	"github.com/brimdata/zq/expr"
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zbuf"
)

// A Merge proc merges multiple upstream inputs into one output.
// If the input streams are ordered according to the configured comparison,
// the output of Merge will have the same order.
type Proc struct {
	parents []proc.Interface
	merger  *zbuf.Merger
}

func New(ctx context.Context, parents []proc.Interface, cmp expr.CompareFn) *Proc {
	pullers := make([]zbuf.Puller, 0, len(parents))
	for _, p := range parents {
		pullers = append(pullers, p)
	}
	return &Proc{
		parents: parents,
		merger:  zbuf.NewMerger(ctx, pullers, cmp),
	}
}

func (m *Proc) Pull() (zbuf.Batch, error) {
	return m.merger.Pull()
}

func (m *Proc) Done() {
	m.merger.Cancel()
	for _, p := range m.parents {
		p.Done()
	}
}
