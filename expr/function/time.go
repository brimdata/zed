package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/pkg/nano"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#now
type Now struct{}

func (n *Now) Call(ctx zed.Allocator, _ []zed.Value) *zed.Value {
	return newTime(ctx, nano.Now())
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#bucket
type Bucket struct{}

func (b *Bucket) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.IsNull() || binArg.IsNull() {
		return zed.NullTime
	}
	var bin nano.Duration
	if binArg.Type == zed.TypeDuration {
		var err error
		bin, err = zed.DecodeDuration(binArg.Bytes)
		if err != nil {
			panic(err)
		}
	} else {
		d, ok := coerce.ToInt(binArg)
		if !ok {
			return newErrorf(ctx, "bucket: second arg must be duration or number")
		}
		bin = nano.Duration(d) * nano.Second
	}
	if zed.TypeUnder(tsArg.Type) == zed.TypeDuration {
		dur, err := zed.DecodeDuration(tsArg.Bytes)
		if err != nil {
			panic(err)
		}
		return newDuration(ctx, dur.Trunc(bin))
	}
	ts, ok := coerce.ToTime(tsArg)
	if !ok {
		return newErrorf(ctx, "bucket: time arg required")
	}
	return newTime(ctx, ts.Trunc(bin))
}
