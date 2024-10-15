package function

import (
	"github.com/brimdata/super"
	"github.com/brimdata/super/order"
	"github.com/brimdata/super/runtime/sam/expr"
)

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#compare
type Compare struct {
	nullsMax, nullsMin expr.CompareFn
	zctx               *zed.Context
}

func NewCompare(zctx *zed.Context) *Compare {
	return &Compare{
		nullsMax: expr.NewValueCompareFn(order.Asc, true),
		nullsMin: expr.NewValueCompareFn(order.Asc, false),
		zctx:     zctx,
	}
}

func (e *Compare) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	nullsMax := true
	if len(args) == 3 {
		if zed.TypeUnder(args[2].Type()) != zed.TypeBool {
			return e.zctx.WrapError("compare: nullsMax arg is not bool", args[2])
		}
		nullsMax = args[2].Bool()
	}
	cmp := e.nullsMax
	if !nullsMax {
		cmp = e.nullsMin
	}
	return zed.NewInt64(int64(cmp(args[0], args[1])))
}
