package expr

//go:generate go run gencomparefuncs.go

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
	"github.com/brimdata/zed/vector"
)

type Compare struct {
	zctx   *zed.Context
	opCode int
	lhs    Evaluator
	rhs    Evaluator
}

func NewCompare(zctx *zed.Context, lhs, rhs Evaluator, op string) *Compare {
	return &Compare{zctx, vector.CompareOpFromString(op), lhs, rhs}
}

func (c *Compare) Eval(val vector.Any) vector.Any {
	return c.eval(c.lhs.Eval(val), c.rhs.Eval(val))
}

func (c *Compare) eval(lhs, rhs vector.Any) vector.Any {
	lhs = vector.Under(lhs)
	rhs = vector.Under(rhs)
	lhs, rhs, _ = coerceVals(c.zctx, lhs, rhs)
	//XXX need to handle overflow (see sam)
	//XXX unions and variants and single-value-with-error variant
	//XXX nulls... for primitives we just do the compare but we need
	// to or the nulls together
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
	f, ok := compareFuncs[vector.FuncCode(c.opCode, kind, lform, rform)]
	if !ok {
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	return f(lhs, rhs)
}
