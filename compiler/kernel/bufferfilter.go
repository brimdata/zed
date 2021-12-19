package kernel

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zson"
	"golang.org/x/text/unicode/norm"
)

// CompileBufferFilter tries to return a BufferFilter for e such that the
// BufferFilter's Eval method returns true for any byte slice containing the ZNG
// encoding of a record matching e. (It may also return true for some byte
// slices that do not match.) compileBufferFilter returns a nil pointer and nil
// error if it cannot construct a useful filter.
func CompileBufferFilter(e dag.Expr) (*expr.BufferFilter, error) {
	switch e := e.(type) {
	case *dag.BinaryExpr:
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
	case *dag.Search:
		if e.Value.Type == "net" || e.Value.Type == "regexp" {
			return nil, nil
		}
		if e.Value.Type == "string" {
			pattern := norm.NFC.Bytes(zed.UnescapeBstring([]byte(e.Value.Text)))
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

func isFieldEqualOrIn(e *dag.BinaryExpr) (*astzed.Primitive, string) {
	if dag.IsRootField(e.LHS) && e.Op == "=" {
		if literal, ok := e.RHS.(*astzed.Primitive); ok {
			return literal, "="
		}
	} else if dag.IsRootField(e.RHS) && e.Op == "in" {
		if literal, ok := e.LHS.(*astzed.Primitive); ok && literal.Type != "net" {
			return literal, "in"
		}
	}
	return nil, ""
}

func xisDollar(e dag.Expr) bool {
	if ref, ok := e.(*dag.Ref); ok && ref.Name == "$" {
		return true
	}
	return false
}

func newBufferFilterForLiteral(l astzed.Primitive) (*expr.BufferFilter, error) {
	switch l.Type {
	case "bool", "byte", "int16", "uint16", "int32", "uint32", "int64", "uint64",
		"float32", "float64", "time", "duration":
		// These are all comparable, so they can require up to three
		// patterns: float, varint, and uvarint.
		return nil, nil
	case "null":
		return nil, nil
	case "regexp":
		// Could try to extract a pattern (e.g., "efg" from "(ab|cd)(efg)+[hi]").
		return nil, nil
	case "string":
		// Match the behavior of zed.ParseLiteral.
		l.Type = "bstring"
	}
	v, err := zson.ParsePrimitive(l.Type, l.Text)
	if err != nil {
		return nil, err
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(v.Encode(nil))
	return expr.NewBufferFilterForString(pattern), nil
}
