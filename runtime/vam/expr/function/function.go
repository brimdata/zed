package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/sam/expr/function"
	"github.com/brimdata/zed/runtime/vam/expr"
)

func New(zctx *zed.Context, name string, narg int) (expr.Function, field.Path, error) {
	argmin := 1
	argmax := 1
	var path field.Path
	var f expr.Function
	switch name {
	case "lower":
		f = &ToLower{zctx}
	default:
		return nil, nil, function.ErrNoSuchFunction
	}
	if err := function.CheckArgCount(narg, argmin, argmax); err != nil {
		return nil, nil, err
	}
	return f, path, nil
}
