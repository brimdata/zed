package kernel

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zson"
	"golang.org/x/text/unicode/norm"
)

// CompileBufferFilter tries to return a BufferFilter for e such that the
// BufferFilter's Eval method returns true for any byte slice containing the ZNG
// encoding of a record matching e. (It may also return true for some byte
// slices that do not match.) compileBufferFilter returns a nil pointer and nil
// error if it cannot construct a useful filter.
func CompileBufferFilter(zctx *zed.Context, e dag.Expr) (*expr.BufferFilter, error) {
	switch e := e.(type) {
	case *dag.BinaryExpr:
		literal, err := isFieldEqualOrIn(zctx, e)
		if err != nil {
			return nil, err
		}
		if literal != nil {
			return newBufferFilterForLiteral(literal)
		}
		if e.Op == "and" {
			left, err := CompileBufferFilter(zctx, e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := CompileBufferFilter(zctx, e.RHS)
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
			left, err := CompileBufferFilter(zctx, e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := CompileBufferFilter(zctx, e.RHS)
			if left == nil || right == nil || err != nil {
				return nil, err
			}
			return expr.NewOrBufferFilter(left, right), nil
		}
		return nil, nil
	case *dag.Search:
		literal, err := zson.ParseValue(zctx, e.Value)
		if err != nil {
			return nil, err
		}
		switch zed.TypeUnder(literal.Type) {
		case zed.TypeNet:
			return nil, nil
		case zed.TypeString:
			pattern := norm.NFC.Bytes(literal.Bytes)
			left := expr.NewBufferFilterForStringCase(string(pattern))
			if left == nil {
				return nil, nil
			}
			right := expr.NewBufferFilterForFieldName(string(pattern))
			return expr.NewOrBufferFilter(left, right), nil
		}
		left := expr.NewBufferFilterForStringCase(e.Text)
		right, err := newBufferFilterForLiteral(literal)
		if left == nil || right == nil || err != nil {
			return nil, err
		}
		return expr.NewOrBufferFilter(left, right), nil
	default:
		return nil, nil
	}
}

// XXX isFieldEqualOrIn should work for any paths not just top-level fields.
// See issue #3412

func isFieldEqualOrIn(zctx *zed.Context, e *dag.BinaryExpr) (*zed.Value, error) {
	if dag.IsTopLevelField(e.LHS) && e.Op == "==" {
		if literal, ok := e.RHS.(*dag.Literal); ok {
			val, err := zson.ParseValue(zctx, literal.Value)
			if err != nil {
				return nil, err
			}
			return val, nil
		}
	} else if dag.IsTopLevelField(e.RHS) && e.Op == "in" {
		if literal, ok := e.LHS.(*dag.Literal); ok {
			val, err := zson.ParseValue(zctx, literal.Value)
			if err != nil {
				return nil, err
			}
			if val.Type == zed.TypeNet {
				return nil, err
			}
			return val, nil
		}
	}
	return nil, nil
}

func newBufferFilterForLiteral(val *zed.Value) (*expr.BufferFilter, error) {
	if id := val.Type.ID(); zed.IsNumber(id) || id == zed.IDNull {
		// All numbers are comparable, so they can require up to three
		// patterns: float, varint, and uvarint.
		return nil, nil
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(val.Encode(nil))
	return expr.NewBufferFilterForString(pattern), nil
}
