package meta

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
)

type pruner struct {
	pred expr.Evaluator
	ectx expr.ResetContext
}

func newPruner(e expr.Evaluator) *pruner {
	return &pruner{pred: e}
}

func (p *pruner) prune(val *zed.Value) bool {
	if p == nil {
		return false
	}
	result := p.pred.Eval(p.ectx.Reset(), val)
	return result.Type == zed.TypeBool && result.Bool()
}
