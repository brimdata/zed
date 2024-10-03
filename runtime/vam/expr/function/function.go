package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/sam/expr/function"
	"github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/vector"
)

func New(zctx *zed.Context, name string, narg int) (expr.Function, field.Path, error) {
	argmin := 1
	argmax := 1
	var path field.Path
	var f expr.Function
	switch name {
	case "base64":
		f = &Base64{zctx}
	case "fields":
		f = NewFields(zctx)
	case "hex":
		f = &Hex{zctx}
	case "join":
		argmax = 2
		f = &Join{zctx: zctx}
	case "kind":
		f = &Kind{zctx: zctx}
	case "len":
		f = &Len{zctx}
	case "levenshtein":
		argmin, argmax = 2, 2
		f = &Levenshtein{zctx}
	case "lower":
		f = &ToLower{zctx}
	case "quiet":
		f = &Quiet{zctx}
	case "replace":
		argmin, argmax = 3, 3
		f = &Replace{zctx}
	case "rune_len":
		f = &RuneLen{zctx}
	case "split":
		argmin, argmax = 2, 2
		f = &Split{zctx}
	case "trim":
		f = &Trim{zctx}
	case "typeof":
		f = &TypeOf{zctx}
	case "upper":
		f = &ToUpper{zctx}
	default:
		return nil, nil, function.ErrNoSuchFunction
	}
	if err := function.CheckArgCount(narg, argmin, argmax); err != nil {
		return nil, nil, err
	}
	return f, path, nil
}

func underAll(args []vector.Any) []vector.Any {
	for i := range args {
		args[i] = vector.Under(args[i])
	}
	return args
}
