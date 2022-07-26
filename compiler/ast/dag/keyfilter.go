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
	lower := append(append([]string{}, prefix...), "lower")
	upper := append(append([]string{}, prefix...), "upper")
	if cropped {
		lower, upper = upper, lower
	}
	return visitLeaves(k.Expr, func(op string, this *This, lit *Literal) Expr {
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
}

func relativeToCompare(op string, lhs, rhs Expr, o order.Which) *BinaryExpr {
	nullsMax := &Literal{"Literal", "false"}
	if o == order.Asc {
		nullsMax.Value = "true"
	}
	lhs = &Call{Kind: "Call", Name: "compare", Args: []Expr{lhs, rhs, nullsMax}}
	return NewBinaryExpr(op, lhs, &Literal{"Literal", "0"})
}

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
		return NewBinaryExpr(e.Op, lhs, rhs)
	case "==", "<", "<=", ">", ">=":
		this, ok := e.LHS.(*This)
		if !ok {
			return nil
		}
		rhs, ok := e.RHS.(*Literal)
		if !ok {
			return nil
		}
		lhs := &This{"This", append([]string{}, this.Path...)}
		return v(e.Op, lhs, rhs)
	default:
		return nil
	}
}
