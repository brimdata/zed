package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#compare
type Compare struct {
	cmp expr.CompareFn
}

func NewCompare() *Compare {
	return &Compare{
		cmp: expr.NewValueCompareFn(true),
	}
}

func (e *Compare) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	return newInt64(ctx, int64(e.cmp(&args[0], &args[1])))
}
