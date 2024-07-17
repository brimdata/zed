package expr

//go:generate go run genarithfuncs.go

import (
	"fmt"

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
	if val, ok := evalForUnionOrVariant(lhs, rhs, a.eval); ok {
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

func evalForUnionOrVariant(lhs, rhs vector.Any, eval func(lhs, rhs vector.Any) vector.Any) (vector.Any, bool) {
	if lhs.Len() != rhs.Len() {
		panic(fmt.Sprintf("mismatched vector lengths: %d vs %d", lhs.Len(), rhs.Len()))
	}

	switch lhs := lhs.(type) {
	case *vector.Variant:
		return evalForUnionOrVariantLHS(lhs.Tags, lhs.TagMap.Reverse, lhs.Values, rhs, eval), true
	case *vector.Union:
		return evalForUnionOrVariantLHS(lhs.Tags, lhs.TagMap.Reverse, lhs.Values, rhs, eval), true
	case *vector.View:
		if lhsVariant, ok := lhs.Any.(*vector.Variant); ok {
			return evalForUnionOrVariantViewLHS(lhs.Index, lhsVariant.Tags, lhsVariant.TagMap.Forward, lhsVariant.Values, rhs, eval), true
		}
		if lhsUnion, ok := lhs.Any.(*vector.Union); ok {
			return evalForUnionOrVariantViewLHS(lhs.Index, lhsUnion.Tags, lhsUnion.TagMap.Forward, lhsUnion.Values, rhs, eval), true
		}
	}

	swapAndEval := func(a, b vector.Any) vector.Any {
		return eval(b, a)
	}
	switch rhs := rhs.(type) {
	case *vector.Variant:
		return evalForUnionOrVariantLHS(rhs.Tags, rhs.TagMap.Reverse, rhs.Values, lhs, swapAndEval), true
	case *vector.Union:
		return evalForUnionOrVariantLHS(rhs.Tags, rhs.TagMap.Reverse, rhs.Values, lhs, swapAndEval), true
	case *vector.View:
		if rhsVariant, ok := rhs.Any.(*vector.Variant); ok {
			return evalForUnionOrVariantViewLHS(rhs.Index, rhsVariant.Tags, rhsVariant.TagMap.Forward, rhsVariant.Values, lhs, swapAndEval), true
		}
		if rhsUnion, ok := rhs.Any.(*vector.Union); ok {
			return evalForUnionOrVariantViewLHS(rhs.Index, rhsUnion.Tags, rhsUnion.TagMap.Forward, rhsUnion.Values, lhs, swapAndEval), true
		}
	}

	return nil, false
}

func evalForUnionOrVariantLHS(lhsTags []uint32, lhsReverse [][]uint32, lhsValues []vector.Any, rhs vector.Any, eval func(lhs, rhs vector.Any) vector.Any) vector.Any {
	results := make([]vector.Any, len(lhsValues))
	for tag, view := range vector.Unstitch(lhsReverse, rhs) {
		results[tag] = eval(lhsValues[tag], view)
	}
	return vector.NewVariant(lhsTags, results)
}

func evalForUnionOrVariantViewLHS(lhsViewIndex []uint32, lhsTags []uint32, lhsForward []uint32, lhsValues []vector.Any, rhs vector.Any, eval func(lhs, rhs vector.Any) vector.Any) vector.Any {
	// Have a view on lhs. Need to convert that to two sets of views. First
	// is a (possibly empty) view per element of lhsValues and second is a
	// corresponding views of rhs. lhsIndexes[k] holds the indexes for the
	// view of lhsValues[k] while rhsIndexes[k] holds indexes for the
	// corresponding view of rhs.
	lhsIndexes := make([][]uint32, len(lhsValues))
	rhsIndexes := make([][]uint32, len(lhsValues))
	// resultTags[k] is the tag for results[k].
	resultTags := make([]uint32, len(lhsViewIndex))
	for k, index := range lhsViewIndex {
		tag := lhsTags[index]
		lhsIndexes[tag] = append(lhsIndexes[tag], lhsForward[index])
		rhsIndexes[tag] = append(rhsIndexes[tag], uint32(k))
		resultTags[k] = tag
	}
	results := make([]vector.Any, len(lhsValues))
	for k := range lhsValues {
		lhsView := vector.NewView2(lhsIndexes[k], lhsValues[k])
		rhsView := vector.NewView2(rhsIndexes[k], rhs)
		results[k] = eval(lhsView, rhsView)
	}
	return vector.NewVariant(resultTags, results)
}
