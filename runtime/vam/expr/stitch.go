package expr

import "github.com/brimdata/zed/vector"

// stitch applies eval to lhs and rhs when either is a union, a variant, a view
// of a union, or a view of a variant. In those cases, it returns a non-nil
// result and true.
func stitch(lhs, rhs vector.Any, eval func(a, b vector.Any) vector.Any) (*vector.Variant, bool) {
	if val, ok := stitchLHS(lhs, rhs, eval); ok {
		return val, true
	}
	swappedEval := func(a, b vector.Any) vector.Any { return eval(b, a) }
	if val, ok := stitchLHS(rhs, lhs, swappedEval); ok {
		return val, true
	}
	return nil, false
}

// stitchLHS is like stitch but only handles the case where lhs is a union, a
// variant, a view of a union, or a view of a variant.
func stitchLHS(lhs, rhs vector.Any, eval func(a, b vector.Any) vector.Any) (*vector.Variant, bool) {
	switch lhs := lhs.(type) {
	case *vector.Union:
		return stitchLHSUnionOrVariant(lhs.Tags, lhs.TagMap.Reverse, lhs.Values, rhs, eval), true
	case *vector.Variant:
		return stitchLHSUnionOrVariant(lhs.Tags, lhs.TagMap.Reverse, lhs.Values, rhs, eval), true
	case *vector.View:
		switch lhsAny := lhs.Any.(type) {
		case *vector.Union:
			return stitchLHSView(lhs.Index, lhsAny.Tags, lhsAny.TagMap.Forward, lhsAny.Values, rhs, eval), true
		case *vector.Variant:
			return stitchLHSView(lhs.Index, lhsAny.Tags, lhsAny.TagMap.Forward, lhsAny.Values, rhs, eval), true
		}
	}
	return nil, false
}

// stitchLHSUnionOrVariant implements stitchLHS when the LHS is a union or
// variant.
func stitchLHSUnionOrVariant(lhsTags []uint32, lhsReverse [][]uint32, lhsValues []vector.Any, rhs vector.Any, eval func(a, b vector.Any) vector.Any) *vector.Variant {
	results := make([]vector.Any, len(lhsValues))
	for k := range lhsValues {
		rhsView := vector.NewView(lhsReverse[k], rhs)
		results[k] = eval(lhsValues[k], rhsView)
	}
	return vector.NewVariant(lhsTags, results)
}

// stitchLHSView implements stitchLHS when the LHS is a view of a union or a
// view of a variant.
func stitchLHSView(lhsViewIndex []uint32, lhsTags []uint32, lhsForward []uint32, lhsValues []vector.Any, rhs vector.Any, eval func(a, b vector.Any) vector.Any) *vector.Variant {
	// Since LHS is a view, we can't just use lhsValues directly as in
	// stitchLHSUnionOrVariant. Instead we have to create views of lhsValues
	// (and, for each of those, a corresonding view of rhs).
	lhsIndexes := make([][]uint32, len(lhsValues))
	rhsIndexes := make([][]uint32, len(lhsValues))
	resultTags := make([]uint32, len(lhsViewIndex))
	for k, index := range lhsViewIndex {
		tag := lhsTags[index]
		lhsIndexes[tag] = append(lhsIndexes[tag], lhsForward[index])
		rhsIndexes[tag] = append(rhsIndexes[tag], uint32(k))
		resultTags[k] = tag
	}
	results := make([]vector.Any, len(lhsValues))
	for k := range lhsValues {
		lhsView := vector.NewView(lhsIndexes[k], lhsValues[k])
		rhsView := vector.NewView(rhsIndexes[k], rhs)
		results[k] = eval(lhsView, rhsView)
	}
	return vector.NewVariant(resultTags, results)
}
