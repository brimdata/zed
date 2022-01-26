package zbuf

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/order"
)

func NewComparator(zctx *zed.Context, layout order.Layout) *expr.Comparator {
	exprs := make([]expr.Evaluator, len(layout.Keys))
	for i, key := range layout.Keys {
		exprs[i] = expr.NewDottedExpr(zctx, key)
	}
	// valueAsBytes establishes a total order.
	exprs = append(exprs, &valueAsBytes{})
	nullsMax := layout.Order == order.Asc
	return expr.NewComparator(nullsMax, !nullsMax, exprs...).WithMissingAsNull()
}

type valueAsBytes struct{}

func (v *valueAsBytes) Eval(_ expr.Context, val *zed.Value) *zed.Value {
	return zed.NewBytes(val.Bytes)
}
