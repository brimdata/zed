package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#compare
type Compare struct {
	nullsMax, nullsMin expr.CompareFn
	zctx               *zed.Context
}

func NewCompare(zctx *zed.Context) *Compare {
	return &Compare{
		nullsMax: expr.NewValueCompareFn(true),
		nullsMin: expr.NewValueCompareFn(false),
		zctx:     zctx,
	}
}

func (e *Compare) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	nullsMax := true
	if len(args) == 3 {
		b := zed.TypeUnder(args[2].Type)
		if b != zed.TypeBool {
			return e.zctx.NewErrorf("expected nullsMax to be of type bool, got %s", zson.FormatType(b))
		}
		nullsMax = zed.DecodeBool(args[2].Bytes)
	}
	cmp := e.nullsMax
	if !nullsMax {
		cmp = e.nullsMin
	}
	return newInt64(ctx, int64(cmp(&args[0], &args[1])))
}
