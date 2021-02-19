package kernel

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

// compileBufferFilter tries to return a BufferFilter for e such that the
// BufferFilter's Eval method returns true for any byte slice containing the ZNG
// encoding of a record matching e. (It may also return true for some byte
// slices that do not match.) compileBufferFilter returns a nil pointer and nil
// error if it cannot construct a useful filter.
func compileBufferFilter(e Expr) (*expr.BufferFilter, error) {
	switch e := e.(type) {
	case *SeqExpr:
		if literal, op, ok := isCompareAny(e); ok && (op == "=" || op == "in") {
			return newBufferFilterForLiteral(literal.Value)
		}
		return nil, nil
	case *BinaryExpr:
		if literal, _ := isFieldEqualOrIn(e); literal != nil {
			return newBufferFilterForLiteral(literal.Value)
		}
		if e.Operator == "and" {
			left, err := compileBufferFilter(e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := compileBufferFilter(e.RHS)
			if err != nil {
				return nil, err
			}
			if left == nil {
				return right, nil
			}
			if right == nil {
				return left, nil
			}
			return expr.NewAndBufferFilter(left, right), nil
		}
		if e.Operator == "or" {
			left, err := compileBufferFilter(e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := compileBufferFilter(e.RHS)
			if left == nil || right == nil || err != nil {
				return nil, err
			}
			return expr.NewOrBufferFilter(left, right), nil
		}
		return nil, nil
	case *SearchExpr:
		typ := zng.AliasedType(e.Value.Type)
		if typ == zng.TypeNet {
			return nil, nil
		}
		if typ == zng.TypeString {
			bytes := e.Value.Bytes
			left := expr.NewBufferFilterForStringCase(string(bytes))
			if left == nil {
				return nil, nil
			}
			right := expr.NewFieldNameFinder(string(bytes))
			return expr.NewOrBufferFilter(left, right), nil
		}
		left := expr.NewBufferFilterForStringCase(e.Text)
		right, err := newBufferFilterForLiteral(e.Value)
		if left == nil || right == nil || err != nil {
			return nil, err
		}
		return expr.NewOrBufferFilter(left, right), nil
	default:
		return nil, nil
	}
}

func isIdOrRoot(e Expr) bool {
	if _, ok := e.(*Identifier); ok {
		return true
	}
	_, ok := e.(*Dot)
	return ok
}

func isRootField(e Expr) bool {
	if isIdOrRoot(e) {
		return true
	}
	b, ok := e.(*BinaryExpr)
	if !ok || b.Operator != "." {
		return false
	}
	if _, ok := b.LHS.(*Dot); !ok {
		return false
	}
	_, ok = b.RHS.(*Identifier)
	return ok
}

func isFieldEqualOrIn(e *BinaryExpr) (*ConstExpr, string) {
	if isRootField(e.LHS) && e.Operator == "=" {
		if literal, ok := e.RHS.(*ConstExpr); ok {
			return literal, "="
		}
	} else if isRootField(e.RHS) && e.Operator == "in" {
		if literal, ok := e.LHS.(*ConstExpr); ok && zng.AliasedType(literal.Value.Type) != zng.TypeNet {
			return literal, "in"
		}
	}
	return nil, ""
}

func isCompareAny(seq *SeqExpr) (*ConstExpr, string, bool) {
	if seq.Name != "or" || len(seq.Methods) != 1 {
		return nil, "", false
	}
	if len(seq.Selectors) != 1 || !isSelectAll(seq.Selectors) {
		return nil, "", false
	}
	m := seq.Methods[0]
	if m.Name != "map" {
		return nil, "", false
	}
	pred, ok := m.Args[0].(*BinaryExpr)
	if !ok {
		return nil, "", false
	}
	if pred.Operator == "=" {
		if !isDollar(pred.LHS) {
			return nil, "", false
		}
		if rhs, ok := pred.RHS.(*ConstExpr); ok && !rhs.IsNet() {
			return rhs, pred.Operator, true
		}
	} else if pred.Operator == "in" {
		if !isDollar(pred.RHS) {
			return nil, "", false
		}
		if lhs, ok := pred.LHS.(*ConstExpr); ok && !lhs.IsNet() {
			return lhs, pred.Operator, true
		}
	}
	return nil, "", false
}

func isSelectAll(selectors []Expr) bool {
	if len(selectors) != 1 {
		return false
	}
	_, ok := selectors[0].(*Dot)
	return ok
}

func isDollar(e Expr) bool {
	id, ok := e.(*Identifier)
	if !ok {
		return false
	}
	return id.Name == "$"
}

func newBufferFilterForLiteral(literal zng.Value) (*expr.BufferFilter, error) {
	// This works for strings, bstrings, type values, and error values.
	// We currently only suport stringy things with this optimization.
	if !zng.IsStringy(literal.Type.ID()) {
		return nil, nil
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(literal.Encode(nil))
	return expr.NewBufferFilterForString(pattern), nil
}
