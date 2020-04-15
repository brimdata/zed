package filter

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zngnative"
)

type Filter func(*zng.Record) bool

func LogicalAnd(left, right Filter) Filter {
	return func(p *zng.Record) bool { return left(p) && right(p) }
}

func LogicalOr(left, right Filter) Filter {
	return func(p *zng.Record) bool { return left(p) || right(p) }
}

func LogicalNot(expr Filter) Filter {
	return func(p *zng.Record) bool { return !expr(p) }
}

func combine(res expr.FieldExprResolver, pred Predicate) Filter {
	return func(r *zng.Record) bool {
		v := res(r)
		if v.Type == nil {
			// field (or sub-field) doesn't exist in this record
			return false
		}
		return pred(v)
	}
}

func CompileFieldCompare(node *ast.CompareField) (Filter, error) {
	literal := node.Value
	// Treat len(field) specially since we're looking at a computed
	// value rather than a field from a record.

	// XXX we need to implement proper expressions
	if op, ok := node.Field.(*ast.FieldCall); ok && op.Fn == "Len" {
		v, err := zng.Parse(literal)
		if err != nil {
			return nil, err
		}
		i, ok := zngnative.CoerceToInt(v)
		if !ok {
			return nil, errors.New("cannot compare len() with non-integer")
		}
		comparison, err := CompareContainerLen(node.Comparator, i)
		if err != nil {
			return nil, err
		}
		resolver, err := expr.CompileFieldExpr(op.Field)
		if err != nil {
			return nil, err
		}
		return combine(resolver, comparison), nil
	}

	comparison, err := Comparison(node.Comparator, literal)
	if err != nil {
		return nil, err
	}
	resolver, err := expr.CompileFieldExpr(node.Field)
	if err != nil {
		return nil, err
	}
	return combine(resolver, comparison), nil
}

func EvalAny(eval Predicate, recursive bool) Filter {
	if !recursive {
		return func(r *zng.Record) bool {
			it := r.ZvalIter()
			for _, c := range r.Type.Columns {
				val, _, err := it.Next()
				if err != nil {
					return false
				}
				if eval(zng.Value{c.Type, val}) {
					return true
				}
			}
			return false
		}
	}

	var fn func(v zcode.Bytes, cols []zng.Column) bool
	fn = func(v zcode.Bytes, cols []zng.Column) bool {
		it := zcode.Iter(v)
		for _, c := range cols {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			recType, isRecord := c.Type.(*zng.TypeRecord)
			if isRecord && fn(val, recType.Columns) {
				return true
			} else if !isRecord && eval(zng.Value{c.Type, val}) {
				return true
			}
		}
		return false
	}
	return func(r *zng.Record) bool {
		return fn(r.Raw, r.Type.Columns)
	}
}

// stringSearchRecord handles the special case of string searching -- it
// matches both field names and values.
func stringSearchRecord(val string, eval Predicate, recursive bool) Filter {
	var match func(v zcode.Bytes, recType *zng.TypeRecord, prefix string) bool
	match = func(v zcode.Bytes, recType *zng.TypeRecord, prefix string) bool {
		it := v.Iter()
		for _, c := range recType.Columns {
			fullname := c.Name
			if len(prefix) > 0 {
				fullname = fmt.Sprintf("%s.%s", prefix, c.Name)
			}
			if stringSearch(fullname, val) {
				return true
			}

			val, _, err := it.Next()
			if err != nil {
				return false
			}
			recType, isRecord := c.Type.(*zng.TypeRecord)
			if isRecord && recursive && match(val, recType, fullname) {
				return true
			} else if !isRecord && eval(zng.Value{c.Type, val}) {
				return true
			}
		}
		return false
	}
	return func(r *zng.Record) bool {
		return match(r.Raw, r.Type, "")
	}
}

func Compile(node ast.BooleanExpr) (Filter, error) {
	switch v := node.(type) {
	case *ast.LogicalNot:
		expr, err := Compile(v.Expr)
		if err != nil {
			return nil, err
		}
		return LogicalNot(expr), nil

	case *ast.LogicalAnd:
		left, err := Compile(v.Left)
		if err != nil {
			return nil, err
		}
		right, err := Compile(v.Right)
		if err != nil {
			return nil, err
		}
		return LogicalAnd(left, right), nil

	case *ast.LogicalOr:
		left, err := Compile(v.Left)
		if err != nil {
			return nil, err
		}
		right, err := Compile(v.Right)
		if err != nil {
			return nil, err
		}
		return LogicalOr(left, right), nil

	case *ast.MatchAll:
		return func(*zng.Record) bool { return true }, nil

	case *ast.CompareField:
		if v.Comparator == "in" {
			resolver, err := expr.CompileFieldExpr(v.Field)
			if err != nil {
				return nil, err
			}
			eql, _ := Comparison("=", v.Value)
			comparison := Contains(eql)
			return combine(resolver, comparison), nil
		}

		return CompileFieldCompare(v)

	case *ast.CompareAny:
		if v.Comparator == "in" {
			compare, err := Comparison("=", v.Value)
			if err != nil {
				return nil, err
			}
			contains := Contains(compare)
			return EvalAny(contains, v.Recursive), nil
		}
		//XXX this is messed up
		if v.Comparator == "searchin" {
			search, err := Comparison("search", v.Value)
			if err != nil {
				return nil, err
			}
			contains := Contains(search)
			return EvalAny(contains, v.Recursive), nil
		}

		comparison, err := Comparison(v.Comparator, v.Value)
		if err != nil {
			return nil, err
		}
		if v.Comparator == "search" && v.Value.Type == "string" {
			return stringSearchRecord(v.Value.Value, comparison, v.Recursive), nil
		}
		return EvalAny(comparison, v.Recursive), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}
