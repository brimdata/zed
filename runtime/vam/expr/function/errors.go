package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#quiet
type Quiet struct {
	zctx *zed.Context
}

func (q *Quiet) Call(args ...vector.Any) vector.Any {
	arg, ok := args[0].(*vector.Error)
	if !ok {
		return args[0]
	}
	if _, ok := arg.Vals.Type().(*zed.TypeOfString); !ok {
		return args[0]
	}
	if c, ok := arg.Vals.(*vector.Const); ok {
		// Fast path
		if s, _ := c.AsString(); s == "missing" {
			return vector.NewStringError(q.zctx, "quiet", c.Len())
		}
		return args[0]
	}
	n := arg.Len()
	vec := vector.NewStringEmpty(n, vector.NewBoolEmpty(n, nil))
	for i := uint32(0); i < n; i++ {
		s, null := vector.StringValue(arg.Vals, i)
		if null {
			vec.Nulls.Set(i)
		}
		if s == "missing" {
			s = "quiet"
		}
		vec.Append(s)
	}
	return vector.NewError(arg.Typ, vec, arg.Nulls)
}
