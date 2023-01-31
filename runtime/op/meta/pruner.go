package meta

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
)

type Pruner struct {
	pred expr.Evaluator
	ectx expr.Context
}

func NewPruner(e expr.Evaluator, o order.Which) *Pruner {
	return &Pruner{
		pred: e,
		ectx: expr.NewContext(),
	}
}

func (p *Pruner) prune(val *zed.Value) bool {
	if p == nil {
		return false
	}
	result := p.pred.Eval(expr.NewContext(), val)
	return result.Type == zed.TypeBool && zed.IsTrue(result.Bytes)
}
