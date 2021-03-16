package kernel

import (
	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

// CompileBufferFilter tries to return a BufferFilter for e such that the
// BufferFilter's Eval method returns true for any byte slice containing the ZNG
// encoding of a record matching e. (It may also return true for some byte
// slices that do not match.) compileBufferFilter returns a nil pointer and nil
// error if it cannot construct a useful filter.
func CompileBufferFilter(e ast.Expr) (*expr.BufferFilter, error) {
	switch e := e.(type) {
	case *ast.SeqExpr:
		if literal, op, ok := isCompareAny(e); ok && (op == "=" || op == "in") {
			return newBufferFilterForLiteral(*literal)
		}
		return nil, nil
	case *ast.BinaryExpr:
		if literal, _ := isFieldEqualOrIn(e); literal != nil {
			return newBufferFilterForLiteral(*literal)
		}
		if e.Op == "and" {
			left, err := CompileBufferFilter(e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := CompileBufferFilter(e.RHS)
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
		if e.Op == "or" {
			left, err := CompileBufferFilter(e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := CompileBufferFilter(e.RHS)
			if left == nil || right == nil || err != nil {
				return nil, err
			}
			return expr.NewOrBufferFilter(left, right), nil
		}
		return nil, nil
	case *ast.Search:
		if e.Value.Type == "net" || e.Value.Type == "regexp" {
			return nil, nil
		}
		if e.Value.Type == "string" {
			pattern, err := zng.TypeBstring.Parse([]byte(e.Value.Value))
			if err != nil {
				return nil, err
			}
			left := expr.NewBufferFilterForStringCase(string(pattern))
			if left == nil {
				return nil, nil
			}
			right := expr.NewFieldNameFinder(string(pattern))
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

func isIdOrRoot(e ast.Expr) bool {
	if _, ok := e.(*ast.Id); ok {
		return true
	}
	_, ok := e.(*ast.Root)
	return ok
}

func isRootField(e ast.Expr) bool {
	if isIdOrRoot(e) {
		return true
	}
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != "." {
		return false
	}
	if _, ok := b.LHS.(*ast.Root); !ok {
		return false
	}
	_, ok = b.RHS.(*ast.Id)
	return ok
}

func isFieldEqualOrIn(e *ast.BinaryExpr) (*ast.Literal, string) {
	if isRootField(e.LHS) && e.Op == "=" {
		if literal, ok := e.RHS.(*ast.Literal); ok {
			return literal, "="
		}
	} else if isRootField(e.RHS) && e.Op == "in" {
		if literal, ok := e.LHS.(*ast.Literal); ok && literal.Type != "net" {
			return literal, "in"
		}
	}
	return nil, ""
}

func isCompareAny(seq *ast.SeqExpr) (*ast.Literal, string, bool) {
	if seq.Name != "or" || len(seq.Methods) != 1 {
		return nil, "", false
	}
	// expression must be a comparison or an in operator
	method := seq.Methods[0]
	if len(method.Args) != 1 || method.Name != "map" {
		return nil, "", false
	}
	pred, ok := method.Args[0].(*ast.BinaryExpr)
	if !ok {
		return nil, "", false
	}
	if pred.Op == "=" {
		if !isDollar(pred.LHS) {
			return nil, "", false
		}
		if rhs, ok := pred.RHS.(*ast.Literal); ok && rhs.Type != "net" {
			return rhs, pred.Op, true
		}
	} else if pred.Op == "in" {
		if !isDollar(pred.RHS) {
			return nil, "", false
		}
		if lhs, ok := pred.LHS.(*ast.Literal); ok && lhs.Type != "net" {
			return lhs, pred.Op, true
		}
	}
	return nil, "", false
}

func isSelectAll(e ast.Expr) bool {
	s, ok := e.(*ast.SelectExpr)
	if !ok || len(s.Selectors) != 1 {
		return false
	}
	_, ok = s.Selectors[0].(*ast.Root)
	return ok
}

func isDollar(e ast.Expr) bool {
	if ref, ok := e.(*ast.Ref); ok && ref.Name == "$" {
		return true
	}
	return false
}

func newBufferFilterForLiteral(l ast.Literal) (*expr.BufferFilter, error) {
	switch l.Type {
	case "bool", "byte", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float64", "time", "duration":
		// These are all comparable, so they can require up to three
		// patterns: float, varint, and uvarint.
		return nil, nil
	case "null":
		return nil, nil
	case "regexp":
		// Could try to extract a pattern (e.g., "efg" from "(ab|cd)(efg)+[hi]").
		return nil, nil
	case "string":
		// Match the behavior of zng.ParseLiteral.
		l.Type = "bstring"
	}
	v, err := zng.Parse(l)
	if err != nil {
		return nil, err
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(v.Encode(nil))
	return expr.NewBufferFilterForString(pattern), nil
}
