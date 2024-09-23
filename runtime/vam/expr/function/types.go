package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typeof
type TypeOf struct {
	zctx *zed.Context
}

func (t *TypeOf) Call(args ...vector.Any) vector.Any {
	val := t.zctx.LookupTypeValue(args[0].Type())
	return vector.NewConst(val, args[0].Len(), nil)
}
