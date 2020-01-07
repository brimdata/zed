package filter

import (
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

type Filter func(*zbuf.Record) bool

func LogicalAnd(left, right Filter) Filter {
	return func(p *zbuf.Record) bool { return left(p) && right(p) }
}

func LogicalOr(left, right Filter) Filter {
	return func(p *zbuf.Record) bool { return left(p) || right(p) }
}

func LogicalNot(expr Filter) Filter {
	return func(p *zbuf.Record) bool { return !expr(p) }
}

func combine(res expr.FieldExprResolver, pred zng.Predicate) Filter {
	return func(r *zbuf.Record) bool {
		v := res(r)
		if v.Type == nil {
			// field (or sub-field) doesn't exist in this record
			return false
		}
		return pred(v)
	}
}

func CompileFieldCompare(node ast.CompareField, val zng.Value) (Filter, error) {
	// Treat len(field) specially since we're looking at a computed
	// value rather than a field from a record.
	if op, ok := node.Field.(*ast.FieldCall); ok && op.Fn == "Len" {
		i, ok := val.(*zng.Int)
		if !ok {
			return nil, errors.New("cannot compare len() with non-integer")
		}
		comparison, err := i.NativeComparison(node.Comparator)
		if err != nil {
			return nil, err
		}
		checklen := func(e zng.TypedEncoding) bool {
			len, err := zng.ContainerLength(e)
			if err != nil {
				return false
			}
			return comparison(int64(len))
		}
		resolver, err := expr.CompileFieldExpr(op.Field)
		if err != nil {
			return nil, err
		}
		return combine(resolver, checklen), nil
	}

	comparison, err := val.Comparison(node.Comparator)
	if err != nil {
		return nil, err
	}
	resolver, err := expr.CompileFieldExpr(node.Field)
	if err != nil {
		return nil, err
	}
	return combine(resolver, comparison), nil
}

func EvalAny(eval zng.Predicate, recursive bool) Filter {
	if !recursive {
		return func(r *zbuf.Record) bool {
			it := r.ZvalIter()
			for _, c := range r.Type.Columns {
				val, _, err := it.Next()
				if err != nil {
					return false
				}
				if eval(zng.TypedEncoding{c.Type, val}) {
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
			} else if !isRecord && eval(zng.TypedEncoding{c.Type, val}) {
				return true
			}
		}
		return false
	}
	return func(r *zbuf.Record) bool {
		return fn(r.Raw, r.Descriptor.Type.Columns)
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

	case *ast.BooleanLiteral:
		return func(p *zbuf.Record) bool { return v.Value }, nil

	case *ast.CompareField:
		z, err := zng.Parse(v.Value)
		if err != nil {
			return nil, err
		}

		if v.Comparator == "in" {
			resolver, err := expr.CompileFieldExpr(v.Field)
			if err != nil {
				return nil, err
			}
			eql, _ := z.Comparison("eql")
			comparison := zng.Contains(eql)
			return combine(resolver, comparison), nil
		}

		return CompileFieldCompare(*v, z)

	case *ast.CompareAny:
		z, err := zng.Parse(v.Value)
		if err != nil {
			return nil, err
		}

		if v.Comparator == "in" {
			eql, _ := z.Comparison("eql")
			comparison := zng.Contains(eql)
			return EvalAny(comparison, v.Recursive), nil
		}
		if v.Comparator == "searchin" {
			search, _ := z.Comparison("search")
			comparison := zng.Contains(search)
			return EvalAny(comparison, v.Recursive), nil
		}

		comparison, err := z.Comparison(v.Comparator)
		if err != nil {
			return nil, err
		}
		return EvalAny(comparison, v.Recursive), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}
