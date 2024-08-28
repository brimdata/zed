package function

import (
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#replace
type Replace struct {
	zctx *zed.Context
}

func (r *Replace) Call(args ...vector.Any) vector.Any {
	args = underAll(args)
	for _, arg := range args {
		if arg.Type() != zed.TypeString {
			return vector.NewWrappedError(r.zctx, "replace: string arg required", arg)
		}
	}
	var errcnt uint32
	sVal := args[0]
	tags := make([]uint32, sVal.Len())
	out := vector.NewStringEmpty(sVal.Len(), vector.NewBoolEmpty(sVal.Len(), nil))
	for i := uint32(0); i < sVal.Len(); i++ {
		s, snull := vector.StringValue(sVal, i)
		old, oldnull := vector.StringValue(args[1], i)
		new, newnull := vector.StringValue(args[2], i)
		if oldnull || newnull {
			tags[i] = 1
			errcnt++
			continue
		}
		if snull {
			out.Nulls.Set(out.Len())
		}
		out.Append(strings.ReplaceAll(s, old, new))
	}
	errval := vector.NewStringError(r.zctx, "replace: an input arg is null", errcnt)
	return vector.NewVariant(tags, []vector.Any{out, errval})
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#lower
type ToLower struct {
	zctx *zed.Context
}

func (t *ToLower) Call(args ...vector.Any) vector.Any {
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
