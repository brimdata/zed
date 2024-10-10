package function

import "github.com/brimdata/zed"

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#coalesce
type Coalesce struct{}

func (c *Coalesce) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	for i := range args {
		val := args[i].Under()
		if !val.IsNull() && !val.IsMissing() && !val.IsQuiet() {
			return args[i]
		}
	}
	return zed.Null
}
