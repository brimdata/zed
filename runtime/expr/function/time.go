package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/coerce"
	"github.com/brimdata/zed/pkg/nano"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#now
type Now struct{}

func (n *Now) Call(ctx zed.Allocator, _ []zed.Value) *zed.Value {
	return newTime(ctx, nano.Now())
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#bucket
type Bucket struct {
	name string
	zctx *zed.Context
}

func (b *Bucket) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.IsNull() || binArg.IsNull() {
		return zed.NullTime
	}
	var bin nano.Duration
	if binArg.Type == zed.TypeDuration {
		bin = zed.DecodeDuration(binArg.Bytes)
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return newErrorf(b.zctx, ctx, "%s: second arg must be duration or number", b)
		}
		bin = nano.Duration(d) * nano.Second
	}
	if zed.TypeUnder(tsArg.Type) == zed.TypeDuration {
		dur := zed.DecodeDuration(tsArg.Bytes)
		return newDuration(ctx, dur.Trunc(bin))
	}
	ts, ok := coerce.ToTime(tsArg)
	if !ok {
		return newErrorf(b.zctx, ctx, "%s: time arg required", b)
	}
	return newTime(ctx, ts.Trunc(bin))
}

func (b *Bucket) String() string {
	if b.name == "" {
		return "bucket"
	}
	return b.name
}
