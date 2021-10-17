package expr

import "github.com/brimdata/zed"

type Var struct {
	ref *zed.Value
}

func NewVar(ref *zed.Value) *Var {
	return &Var{ref}
}

func (v *Var) Eval(*zed.Value) (zed.Value, error) {
	zv := *v.ref
	if zv.Type == nil {
		return zed.Value{}, zed.ErrMissing
	}
	return zv, nil
}
