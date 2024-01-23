package zbuf

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/sam/expr"
)

func NewComparator(zctx *zed.Context, sortKey order.SortKey) *expr.Comparator {
	exprs := make([]expr.Evaluator, len(sortKey.Keys))
	for i, key := range sortKey.Keys {
		exprs[i] = expr.NewDottedExpr(zctx, key)
	}
	// valueAsBytes establishes a total order.
	exprs = append(exprs, &valueAsBytes{})
	nullsMax := sortKey.Order == order.Asc
	return expr.NewComparator(nullsMax, !nullsMax, exprs...).WithMissingAsNull()
}

func NewComparatorNullsMax(zctx *zed.Context, sortKey order.SortKey) *expr.Comparator {
	exprs := make([]expr.Evaluator, len(sortKey.Keys))
	for i, key := range sortKey.Keys {
		exprs[i] = expr.NewDottedExpr(zctx, key)
	}
	// valueAsBytes establishes a total order.
	exprs = append(exprs, &valueAsBytes{})
	reverse := sortKey.Order == order.Desc
	return expr.NewComparator(true, reverse, exprs...).WithMissingAsNull()
}

type valueAsBytes struct{}

func (v *valueAsBytes) Eval(ectx expr.Context, val zed.Value) zed.Value {
	return zed.NewBytes(val.Bytes())
}
