package expr

//go:generate go run gencomparefuncs.go

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
	"github.com/brimdata/zed/vector"
)

type Compare struct {
	zctx *zed.Context
	op   string
	lhs  Evaluator
	rhs  Evaluator
}

func NewCompare(zctx *zed.Context, lhs, rhs Evaluator, operator string) *Compare {
	return &Compare{zctx, operator, lhs, rhs}
}

func (c *Compare) Eval(val vector.Any) vector.Any {
	return c.compare(c.lhs.Eval(val), c.rhs.Eval(val))
}

func (c *Compare) compare(lhs, rhs vector.Any) vector.Any {
	lhs = vector.Under(lhs)
	rhs = vector.Under(rhs)
	lhs, rhs, _ = coerceVals(c.zctx, lhs, rhs)
	//XXX need to handle overflow (see sam)
	//XXX unions and variants and single-value-with-error variant
	//XXX nulls... for primitives we just do the compare but we need
	// to or the nulls together
	if lc, ok := lhs.(*vector.Const); ok {
		if rc, ok := rhs.(*vector.Const); ok {
			return compareConsts(c.op, lc, rc)
		}
	}
	kind := vector.KindOf(lhs)
	if kind != vector.KindOf(rhs) {
		panic("vector kind mismatch after coerce")
	}
	lform, ok := vector.FormOf(lhs)
	if !ok {
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	rform, ok := vector.FormOf(rhs)
	if !ok {
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	compare, ok := compareFuncs[vector.CompareOpCode(c.op, kind, lform, rform)]
	if !ok {
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	return compare(lhs, rhs)
}

func compareConsts(op string, lhs, rhs *vector.Const) vector.Any {
	compare := expr.NewValueCompareFn(order.Asc, false)
	compareResult := compare(lhs.Value(), rhs.Value())
	var result bool
	switch op {
	case "==":
		result = compareResult == 0
	case "!=":
		result = compareResult != 0
	case "<":
		result = compareResult < 0
	case "<=":
		result = compareResult <= 0
	case ">":
		result = compareResult > 0
	case ">=":
		result = compareResult >= 0
	default:
		panic(op)
	}
	return vector.NewConst(nil, zed.NewBool(result), lhs.Len(), nil)
}
