package expr

import "github.com/brimdata/zed"

type Var int

var _ Evaluator = (*Var)(nil)

func NewVar(slot int) *Var {
	return (*Var)(&slot)
}

func (v Var) Eval(ectx Context, _ *zed.Value) *zed.Value {
	vars := ectx.Vars()
	return &vars[v]
}
