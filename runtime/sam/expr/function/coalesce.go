package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#coalesce
type Coalesce struct{}

func (c *Coalesce) Call(_ expr.Context, args []zed.Value) zed.Value {
	for i := range args {
		val := &args[i]
		if !val.IsNull() && !val.IsMissing() && !val.IsQuiet() {
			return *val
		}
	}
	return zed.Null
}
