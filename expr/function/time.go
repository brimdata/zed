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

//XXX this name isn't right

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#trunc
type Trunc struct{}

func (t *Trunc) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	tsArg := args[0]
	binArg := args[1]
	if tsArg.IsNull() || binArg.IsNull() {
		return zed.NullTime
	}
	ts, ok := coerce.ToTime(tsArg)
	if !ok {
		return newErrorf(ctx, "trunc: time arg required")
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
			return newErrorf(ctx, "trunc: second arg must be duration or number")
		}
		bin = nano.Duration(d) * nano.Second
	}
	return newTime(ctx, nano.Ts(ts.Trunc(bin)))
}
