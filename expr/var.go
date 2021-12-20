package expr

import "github.com/brimdata/zed"

type Var struct {
	ref *zed.Value
}

var _ Evaluator = (*Var)(nil)

func NewVar(ref *zed.Value) *Var {
	return &Var{ref}
}

func (v *Var) Eval(Context, *zed.Value) *zed.Value {
	val := v.ref
	if val == nil || val.Type == nil {
		return zed.Missing
	}
	return val
}
