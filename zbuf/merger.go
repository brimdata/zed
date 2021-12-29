package zbuf

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/order"
)

func NewCompareFn(layout order.Layout) expr.CompareFn {
	nullsMax := layout.Order == order.Asc
	exprs := make([]expr.Evaluator, len(layout.Keys))
	for i, key := range layout.Keys {
		exprs[i] = expr.NewDottedExpr(key)
	}
	fn := expr.NewCompareFn(nullsMax, exprs...)
	fn = totalOrderCompare(fn)
	if layout.Order == order.Asc {
		return fn
	}
	return func(a, b *zed.Value) int { return fn(b, a) }
}

func totalOrderCompare(fn expr.CompareFn) expr.CompareFn {
	return func(a, b *zed.Value) int {
		cmp := fn(a, b)
		if cmp == 0 {
			return bytes.Compare(a.Bytes, b.Bytes)
		}
		return cmp
	}
}
