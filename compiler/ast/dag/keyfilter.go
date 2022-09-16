package dag

import (
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"golang.org/x/exp/slices"
)

type KeyFilter struct {
	Expr Expr
}

// NewKeyFilter creates a KeyFilter that contains a modified form of node where
// only predicates operating on key are kept. The underlying expression is
// rewritten in a manner so that results produced by the filter will always be
// a superset of the results produced by the parent filter (i.e., it will not
// filter values that would not be not also filtered by the original filter).
// Currently KeyFilter only recognizes simple key predicates against a literal
// value and using the comparators ==, >=, >, <, and <=; otherwise the predicate is
// ignored.
func NewKeyFilter(key field.Path, node Expr) *KeyFilter {
	e, _ := visitLeaves(node, func(cmp string, lhs *This, rhs *Literal) Expr {
		if !key.Equal(lhs.Path) {
			return nil
		}
		return NewBinaryExpr(cmp, lhs, rhs)
	})
	if e == nil {
		return nil
	}
	return &KeyFilter{e}
}

func (k *KeyFilter) CroppedByExpr(o order.Which, prefix ...string) Expr {
	return k.newExpr(o, prefix, true)
}

// SpanExpr creates an Expr that returns true if the KeyFilter has a value
// within a span of values. The span compared against must be a record with the
// fields "lower" and "upper", where "lower" is the inclusive lower bounder and
// "upper" is the exclusive upper bound.
func (k *KeyFilter) SpanExpr(o order.Which, prefix ...string) Expr {
	return k.newExpr(o, prefix, false)
}

func (k *KeyFilter) newExpr(o order.Which, prefix []string, cropped bool) Expr {
	lower := append(slices.Clone(prefix), "lower")
	upper := append(slices.Clone(prefix), "upper")
	if cropped {
		lower, upper = upper, lower
	}
	e, _ := visitLeaves(k.Expr, func(op string, this *This, lit *Literal) Expr {
		switch op {
		case "==":
			if cropped {
				return &Literal{"Literal", "false"}
			}
			lhs := relativeToCompare("<=", &This{"This", lower}, lit, o)
			rhs := relativeToCompare(">=", &This{"This", upper}, lit, o)
			return NewBinaryExpr("and", lhs, rhs)
		case "<", "<=":
			this.Path = lower
		case ">", ">=":
			this.Path = upper
		}
		return relativeToCompare(op, this, lit, o)
	})
	return e
}

func relativeToCompare(op string, lhs, rhs Expr, o order.Which) *BinaryExpr {
	nullsMax := &Literal{"Literal", "false"}
	if o == order.Asc {
		nullsMax.Value = "true"
	}
	lhs = &Call{Kind: "Call", Name: "compare", Args: []Expr{lhs, rhs, nullsMax}}
	return NewBinaryExpr(op, lhs, &Literal{"Literal", "0"})
}

func visitLeaves(node Expr, v func(cmp string, lhs *This, rhs *Literal) Expr) (Expr, bool) {
	e, ok := node.(*BinaryExpr)
	if !ok {
		return nil, true
	}
	switch e.Op {
	case "and", "or":
		lhs, lok := visitLeaves(e.LHS, v)
		rhs, rok := visitLeaves(e.RHS, v)
		if !lok || !rok {
			return nil, false
		}
		if lhs == nil {
			if e.Op == "or" {
				return nil, false
			}
			return rhs, e.Op != "or"
		}
		if rhs == nil {
			if e.Op == "or" {
				return nil, false
			}
			return lhs, true
		}
		return NewBinaryExpr(e.Op, lhs, rhs), true
	case "==", "<", "<=", ">", ">=":
		this, ok := e.LHS.(*This)
		if !ok {
			return nil, true
		}
		rhs, ok := e.RHS.(*Literal)
		if !ok {
			return nil, true
		}
		lhs := &This{"This", slices.Clone(this.Path)}
		return v(e.Op, lhs, rhs), true
	default:
		return nil, true
	}
}
