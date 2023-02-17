package custom

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent zbuf.Puller
	name   string
	expr   expr.Evaluator
}

func New(parent zbuf.Puller, name string, e expr.Evaluator) *Proc {
	return &Proc{
		parent: parent,
		name:   name,
		expr:   e,
	}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	if done {
		return p.parent.Pull(true)
	}
	batch, err := p.parent.Pull(false)
	if batch == nil || err != nil {
		return nil, err
	}
	vals := make([]zed.Value, len(batch.Values()))
	for i, v := range batch.Values() {
		vals[i] = *p.expr.Eval(batch, &v)
	}
	return zbuf.NewBatch(batch, vals), nil
}
