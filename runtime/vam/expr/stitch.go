package expr

import "github.com/brimdata/zed/vector"

// stitch applies eval to lhs and rhs when either is a union or variant. In
// those cases, it returns a non-nil result and true.
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

// stitchLHS is like stitch but only handles the case where lhs is a union or
// variant.
func stitchLHS(lhs, rhs vector.Any, eval func(a, b vector.Any) vector.Any) (*vector.Variant, bool) {
	var lhsVariant *vector.Variant
	switch lhs := lhs.(type) {
	case *vector.Union:
		lhsVariant = lhs.Variant
	case *vector.Variant:
		lhsVariant = lhs
	default:
		return nil, false
	}

	reverse := lhsVariant.TagMap.Reverse
	results := make([]vector.Any, len(lhsVariant.Values))
	for k, lhs := range lhsVariant.Values {
		rhsView := vector.NewView(reverse[k], rhs)
		results[k] = eval(lhs, rhsView)
	}
	return vector.NewVariant(lhsVariant.Tags, results), true
}
