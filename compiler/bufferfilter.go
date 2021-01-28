package compiler

import (
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

// compileBufferFilter tries to return a BufferFilter for e such that the
// BufferFilter's Eval method returns true for any byte slice containing the ZNG
// encoding of a record matching e. (It may also return true for some byte
// slices that do not match.) compileBufferFilter returns a nil pointer and nil
// error if it cannot construct a useful filter.
func compileBufferFilter(e ast.BooleanExpr) (*expr.BufferFilter, error) {
	switch e := e.(type) {
	case *ast.CompareAny:
		if e.Comparator != "=" && e.Comparator != "in" {
			return nil, nil
		}
		return newBufferFilterForLiteral(e.Value)
	case *ast.CompareField:
		if e.Comparator != "=" && e.Comparator != "in" {
			return nil, nil
		}
		return newBufferFilterForLiteral(e.Value)
	case *ast.BinaryExpression:
		return nil, nil
	case *ast.LogicalAnd:
		left, err := compileBufferFilter(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := compileBufferFilter(e.Right)
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
	case *ast.LogicalOr:
		left, err := compileBufferFilter(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := compileBufferFilter(e.Right)
		if left == nil || right == nil || err != nil {
			return nil, err
		}
		return expr.NewOrBufferFilter(left, right), nil
	case *ast.LogicalNot, *ast.MatchAll:
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
		return nil, fmt.Errorf("filter AST unknown type: %T", e)
	}
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
