package function

import (
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#lower
type ToLower struct {
	zctx *zed.Context
}

func (t *ToLower) Call(args []vector.Any) vector.Any {
	v := vector.Under(args[0])
	if v.Type() != zed.TypeString {
		return vector.NewWrappedError(t.zctx, "lower: string arg required", v)
	}
	out := vector.NewStringEmpty(v.Len(), vector.NewBoolEmpty(v.Len(), nil))
	for i := uint32(0); i < v.Len(); i++ {
		s, null := vector.StringValue(v, i)
		if null {
			out.Nulls.Set(i)
		}
		out.Append(strings.ToLower(s))
	}
	return out
}
