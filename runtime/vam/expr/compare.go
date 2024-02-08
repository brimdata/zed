package expr

import (
	"fmt"

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

func (c *Compare) compare(l, r vector.Any) vector.Any {
	l = vector.Under(l)
	r = vector.Under(r)
	//XXX abstract this into a generic stitch operator with callback
	if u, ok := l.(*vector.Union); ok {
		results := make([]vector.Any, len(u.Values))
		for tag, view := range u.Unstitch(r) {
			results[tag] = c.compare(u.Values[tag], view)
		}
		return u.Stitch(c.zctx, results)
	}
	lhs, rhs, _ := coerceVals(c.zctx, l, r)
	//XXX error?
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
		//XXX better error message
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	rform, ok := vector.FormOf(rhs)
	if !ok {
		//XXX better error message
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	compareFunc, ok := compareFuncs[vector.CompareOpCode(c.op, kind, lform, rform)]
	if !ok {
		//XXX better error message
		return vector.NewStringError(c.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	//XXX need to handle overflow (see sam)
	//XXX unions and variants and single-value-with-error variant
	//XXX nulls... for primitives we just do the compare but we need
	// to or the nulls together
	return compareFunc(lhs, rhs)
}

//XXX if we have a view of a union on one side and a vector of scalar values
// on the other side, it might be the case that the underlying type of the viewed
// elements all has the same type as the view of the union (in other words the
// elements of the view of the union are all one type).  OR... two unions could
// be type-compatible element by element... is this worth checking?  feels like
// a later optimization.

// stitch is a set of uniformly typed vectors that when blended form
// a union type, but the stitch separates the vectors like dense-union

// XXX unused
func derefView(vals vector.Any) (vector.Any, []uint32) {
	var idx []uint32
	for {
		switch v := vals.(type) {
		case *vector.Dict:
			vals = v.Any
		case *vector.View:
			idx = v.Index
			vals = v.Any
		default:
			return vals, idx
		}
	}
}

func swapOp(op string) string {
	switch op {
	case "<":
		return ">"
	case "<=":
		return ">="
	case ">":
		return "<"
	case ">=":
		return "<="
	default:
		return op

	}
}

func compareConsts(op string, lhs, rhs *vector.Const) vector.Any {
	l := lhs.Value()
	r := rhs.Value()
	cmp := expr.NewValueCompareFn(order.Asc, false)
	var result bool
	switch op {
	case "==":
		result = cmp(l, r) == 0
	case "!=":
		result = cmp(l, r) != 0
	case "<":
		result = cmp(l, r) < 0
	case "<=":
		result = cmp(l, r) <= 0
	case ">":
		result = cmp(l, r) > 0
	case ">=":
		result = cmp(l, r) >= 0
	default:
		panic(fmt.Sprintf("unknown op %q", op))
	}
	return vector.NewConst(zed.NewBool(result), lhs.Len(), nil)
}
