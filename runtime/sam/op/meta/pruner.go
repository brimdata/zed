package meta

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
)

type pruner struct {
	pred expr.Evaluator
	ectx expr.Context
}

func newPruner(e expr.Evaluator) *pruner {
	return &pruner{
		pred: e,
		ectx: expr.NewContext(zed.NewArena()),
	}
}

func (p *pruner) prune(val zed.Value) bool {
	if p == nil {
		return false
	}
	p.ectx.Arena().Reset()
	result := p.pred.Eval(p.ectx, val)
	return result.Type() == zed.TypeBool && result.Bool()
}
