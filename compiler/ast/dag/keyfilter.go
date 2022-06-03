package dag

import (
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
)

type KeyFilter struct {
	Expr Expr
}

// NewKeyFilter creates a KeyFilter that contains an modified form of node where
// only predicates operating on key are kept. The underlying expression is
// rewritten in a manner so that results produced by the filter will always be
// a superset of the results produced by the parent filter (i.e., it will not
// filter values that would not be not also filtered by the original filter).
// Currently KeyFilter only recognizes simple key predicates against a literal
// value and using the comparators ==, >=, >, <, <=, otherwise the predicate is
// ignored.
func NewKeyFilter(key field.Path, node Expr) *KeyFilter {
	e := visitLeaves(node, func(cmp string, lhs *This, rhs *Literal) Expr {
		if !key.Equal(lhs.Path) {
			return nil
		}
		return &BinaryExpr{
			Op:  cmp,
			LHS: lhs,
			RHS: rhs,
		}
	})
	if e == nil {
		return nil
	}
	return &KeyFilter{e}
}

// SpanFilter creates an Expr that returns true if the KeyFilter has a value
// within a span of values. The span compared against must be a record with the
// fields "lower" and "upper", where "lower" is the inclusive lower bounder and
// "upper" is the exclusive upper bound.
func (k *KeyFilter) SpanFilter(o order.Which, prefix ...string) Expr {
	lower := append([]string{}, append(prefix, "lower")...)
	upper := append([]string{}, append(prefix, "upper")...)
	return k.VisitLeaves(func(cmp string, this *This, val *Literal) Expr {
		switch cmp {
		case "==":
			return &BinaryExpr{
				Op:  "and",
				LHS: relativeToCompare("<=", &This{Path: lower}, val, o),
				RHS: relativeToCompare(">=", &This{Path: upper}, val, o),
			}
		case "<", "<=":
			this.Path = lower
		case ">", ">=":
			this.Path = upper
		}
		return relativeToCompare(cmp, this, val, o)
	})
}

// CroppedByFilter produces an expression which returns true if a span known
// to overlap the key filter is cropped by filter (i.e., not all values in the
// span evaluate true against the KeyFilter.
func (k *KeyFilter) CroppedByFilter(o order.Which, prefix ...string) Expr {
	lower := append([]string{}, append(prefix, "lower")...)
	upper := append([]string{}, append(prefix, "upper")...)
	return k.VisitLeaves(func(cmp string, this *This, val *Literal) Expr {
		switch cmp {
		case "==":
			return &Literal{Value: "false"}
		case "<", "<=":
			this.Path = upper
		case ">", ">=":
			this.Path = lower
		}
		return relativeToCompare(cmp, this, val, o)
	})
}

func (k *KeyFilter) VisitLeaves(visit visitLeaf) Expr {
	return visitLeaves(k.Expr, visit)
}

func relativeToCompare(op string, lhs, rhs Expr, o order.Which) *BinaryExpr {
	nullsMax := &Literal{Value: "false"}
	if o == order.Asc {
		nullsMax.Value = "true"
	}
	return &BinaryExpr{
		Op: op,
		LHS: &Call{
			Name: "compare",
			Args: []Expr{lhs, rhs, nullsMax},
		},
		RHS: &Literal{Value: "0"},
	}
}

type visitLeaf func(cmp string, lhs *This, rhs *Literal) Expr

func visitLeaves(node Expr, v func(cmp string, lhs *This, rhs *Literal) Expr) Expr {
	e, ok := node.(*BinaryExpr)
	if !ok {
		return nil
	}
	switch e.Op {
	case "or", "and":
		lhs := visitLeaves(e.LHS, v)
		rhs := visitLeaves(e.RHS, v)
		if lhs == nil {
			return rhs
		}
		if rhs == nil {
			return lhs
		}
		return &BinaryExpr{
			Op:  e.Op,
			LHS: lhs,
			RHS: rhs,
		}
	case "==", "<", "<=", ">", ">=":
		this, ok := e.LHS.(*This)
		if !ok {
			return nil
		}
		rhs, ok := e.RHS.(*Literal)
		if !ok {
			return nil
		}
		// Copy this.
		var lhs This
		lhs.Path = append(lhs.Path, this.Path...)
		return v(e.Op, &lhs, rhs)
	default:
		return nil
	}
}
