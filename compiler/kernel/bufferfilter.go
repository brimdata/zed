package kernel

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
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
		if literal, _ := isFieldEqualOrIn(zctx, e); literal != nil {
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
		val, err := zson.ParseValue(zctx, e.Value)
		if err != nil {
			return nil, err
		}
		if val.Type == zed.TypeNet {
			return nil, nil
		}
		if val.Type == zed.TypeString {
			pattern := norm.NFC.Bytes(zed.UnescapeBstring(val.Bytes))
			left := expr.NewBufferFilterForStringCase(string(pattern))
			if left == nil {
				return nil, nil
			}
			right := expr.NewFieldNameFinder(string(pattern))
			return expr.NewOrBufferFilter(left, right), nil
		}
		left := expr.NewBufferFilterForStringCase(e.Text)
		literal, err := zson.ParseValue(zctx, e.Value)
		if err != nil {
			return nil, err
		}
		right, err := newBufferFilterForLiteral(&literal)
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

func isFieldEqualOrIn(zctx *zed.Context, e *dag.BinaryExpr) (*zed.Value, string) {
	if dag.IsTopLevelField(e.LHS) && e.Op == "=" {
		if literal, ok := e.RHS.(*dag.Literal); ok {
			val, err := zson.ParseValue(zctx, literal.Value)
			if err != nil {
				return nil, ""
			}
			return &val, "="
		}
	} else if dag.IsTopLevelField(e.RHS) && e.Op == "in" {
		if literal, ok := e.LHS.(*dag.Literal); ok {
			val, err := zson.ParseValue(zctx, literal.Value)
			if err != nil {
				return nil, ""
			}
			if val.Type == zed.TypeNet {
				return nil, ""
			}
			return &val, "="
		}
	}
	return nil, ""
}

func newBufferFilterForLiteral(val *zed.Value) (*expr.BufferFilter, error) {
	switch val.Type.(type) {
	case *zed.TypeOfBool, *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfUint16, *zed.TypeOfInt32, *zed.TypeOfUint32, *zed.TypeOfInt64, *zed.TypeOfUint64, *zed.TypeOfFloat32, *zed.TypeOfFloat64, *zed.TypeOfTime, *zed.TypeOfDuration:
		// These are all comparable, so they can require up to three
		// patterns: float, varint, and uvarint.
		return nil, nil
	case *zed.TypeOfNull:
		return nil, nil
	case *zed.TypeOfString:
		// Match the behavior of zed.ParseLiteral.
		val = zed.NewValue(zed.TypeBstring, val.Bytes)
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(val.Encode(nil))
	return expr.NewBufferFilterForString(pattern), nil
}
