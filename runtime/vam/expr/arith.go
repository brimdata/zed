package expr

//go:generate go run genarithfuncs.go

import (
	"fmt"
	"slices"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
	"github.com/brimdata/zed/vector"
)

type Arith struct {
	zctx   *zed.Context
	opCode int
	lhs    Evaluator
	rhs    Evaluator
}

func NewArith(zctx *zed.Context, lhs, rhs Evaluator, op string) *Arith {
	return &Arith{zctx, vector.ArithOpFromString(op), lhs, rhs}
}

func (a *Arith) Eval(val vector.Any) vector.Any {
	return a.eval(a.lhs.Eval(val), a.rhs.Eval(val))
}

func (a *Arith) eval(lhs, rhs vector.Any) vector.Any {
	lhs = vector.Under(lhs)
	rhs = vector.Under(rhs)
	if val, ok := applyToUnion(a.zctx, lhs, rhs, a.eval); ok {
		return val
	}
	lhs, rhs, errVal := coerceVals(a.zctx, lhs, rhs)
	if errVal != nil {
		return errVal
	}
	kind := vector.KindOf(lhs)
	if kind != vector.KindOf(rhs) {
		panic(fmt.Sprintf("vector kind mismatch after coerce (%#v and %#v)", lhs, rhs))
	}
	lform, ok := vector.FormOf(lhs)
	if !ok {
		return vector.NewStringError(a.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	rform, ok := vector.FormOf(rhs)
	if !ok {
		return vector.NewStringError(a.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	f, ok := arithFuncs[vector.FuncCode(a.opCode, kind, lform, rform)]
	if !ok {
		return vector.NewStringError(a.zctx, coerce.ErrIncompatibleTypes.Error(), lhs.Len())
	}
	return f(lhs, rhs)
}

func applyToUnion(zctx *zed.Context, lhs, rhs vector.Any, eval func(lhs, rhs vector.Any) vector.Any) (vector.Any, bool) {
	if lhs.Len() != rhs.Len() {
		panic(fmt.Sprintf("mismatched vector lengths: %d vs %d", lhs.Len(), rhs.Len()))
	}
	if lhs, ok := lhs.(*vector.Union); ok {
		results := make([]vector.Any, len(lhs.Values))
		for tag, view := range lhs.Unstitch(rhs) {
			results[tag] = eval(lhs.Values[tag], view)
		}
		return lhs.Stitch(zctx, results), true
	}
	if rhs, ok := rhs.(*vector.Union); ok {
		results := make([]vector.Any, len(rhs.Values))
		for tag, view := range rhs.Unstitch(lhs) {
			results[tag] = eval(view, rhs.Values[tag])
		}
		return rhs.Stitch(zctx, results), true
	}

	if lhsView, ok := lhs.(*vector.View); ok {
		if lhsUnion, ok := lhsView.Any.(*vector.Union); ok {
			// Have a view on lhsUnion. Need to convert that to two
			// sets of views. First has a view per element of
			// lhsUnion.Values. Second has a corresponding view of
			// rhs.
			reverse := lhsUnion.TagMap.Reverse
			// viewIndexes[k] will hold the view indexes for XXX.
			viewIndexes := make([][]uint32, len(lhsUnion.Values))
			// resultTags[k] is the union tag for the k-th element of the result vector.
			resultTags := make([]uint32, lhsView.Len())
			for k, index := range lhsView.Index {
				tag := lhsUnion.Tags[index]
				resultTags[k] = tag
				unionValuesIndex, ok := slices.BinarySearch(reverse[tag], index)
				if !ok {
					panic("index not in reverse")
				}
				viewIndexes[tag] = append(viewIndexes[tag], uint32(unionValuesIndex))
			}
			// lhsViews[k] will hold the view for lhsUnion.Values[k].
			lhsViews := make([]vector.Any, len(lhsUnion.Values))
			for k := range lhsViews {
				lhsViews[k] = vector.NewView2(viewIndexes[k], lhsUnion.Values[k])
			}
			// No need to allocate another slice for results.
			results := lhsViews
			for tag, lhsView := range lhsViews {
				rhsView := vector.NewView2(reverse[tag], rhs)
				results[tag] = eval(lhsView, rhsView)
			}
			// XXX Need to merge elements of results that have the same type.
			return vector.Stitch(zctx, resultTags, results), true
		}
	}

	if rhsView, ok := rhs.(*vector.View); ok {
		if rhsUnion, ok := rhsView.Any.(*vector.Union); ok {
			reverse := rhsUnion.TagMap.Reverse
			viewIndexes := make([][]uint32, len(rhsUnion.Values))
			resultTags := make([]uint32, rhsView.Len())
			for k, index := range rhsView.Index {
				tag := rhsUnion.Tags[index]
				resultTags[k] = tag
				unionValuesIndex, ok := slices.BinarySearch(reverse[tag], index)
				if !ok {
					panic("index not in reverse")
				}
				viewIndexes[tag] = append(viewIndexes[tag], uint32(unionValuesIndex))
			}
			rhsViews := make([]vector.Any, len(rhsUnion.Values))
			for k := range rhsViews {
				rhsViews[k] = vector.NewView2(viewIndexes[k], rhsUnion.Values[k])
			}
			// No need to allocate another slice for results.
			results := rhsViews
			for tag, rhsView := range rhsViews {
				lhsView := vector.NewView2(viewIndexes[tag], lhs)
				results[tag] = eval(lhsView, rhsView)
			}
			// XXX Need to merge elements of results that have the same type.
			return vector.Stitch(zctx, resultTags, results), true
		}
	}

	return nil, false
}
