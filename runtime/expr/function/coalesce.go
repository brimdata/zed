package function

import "github.com/brimdata/zed"

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#coalesce
type Coalesce struct{}

func (c *Coalesce) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	for i := range args {
		val := &args[i]
		if !val.IsNull() && !val.IsMissing() && !val.IsQuiet() {
			return ctx.CopyValue(val)
		}
	}
	return zed.Null
}
