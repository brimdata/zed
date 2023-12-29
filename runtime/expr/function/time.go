package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/coerce"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#now
type Now struct{}

func (n *Now) Call(ctx zed.Allocator, _ []zed.Value) *zed.Value {
	return ctx.CopyValue(*zed.NewTime(nano.Now()))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#bucket
type Bucket struct {
	name string
	zctx *zed.Context
}

func (b *Bucket) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	tsArg := &args[0]
	binArg := &args[1]
	if tsArg.IsNull() || binArg.IsNull() {
		return zed.NullTime
	}
	var bin nano.Duration
	if binArg.Type() == zed.TypeDuration {
		bin = nano.Duration(binArg.Int())
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return wrapError(b.zctx, ctx, b.name+": second argument is not a duration or number", binArg)
		}
		bin = nano.Duration(d) * nano.Second
	}
	if zed.TypeUnder(tsArg.Type()) == zed.TypeDuration {
		dur := nano.Duration(tsArg.Int())
		return ctx.CopyValue(*zed.NewDuration(dur.Trunc(bin)))
	}
	v, ok := coerce.ToInt(tsArg)
	if !ok {
		return wrapError(b.zctx, ctx, b.name+": first argument is not a time", tsArg)
	}
	return ctx.CopyValue(*zed.NewTime(nano.Ts(v).Trunc(bin)))
}
