package filter

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zval"
)

type Filter func(*zson.Record) bool

func LogicalAnd(left, right Filter) Filter {
	return func(p *zson.Record) bool { return left(p) && right(p) }
}

func LogicalOr(left, right Filter) Filter {
	return func(p *zson.Record) bool { return left(p) || right(p) }
}

func LogicalNot(expr Filter) Filter {
	return func(p *zson.Record) bool { return !expr(p) }
}

func SearchString(s string) Filter {
	pattern := []byte(s)
	return func(p *zson.Record) bool {
		// Go implements a very efficient string search algorithm so we
		// use it here first to rule out misses on a substring match.
		if !bytes.Contains(p.Raw, pattern) {
			return false
		}
		// If we have a hit, double check field by field in case the
		// framing bytes give us a false positive.
		// XXX we should refactor these iterators to make this tighter.
		it := p.ZvalIter()
		for _, c := range p.Type.Columns {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			switch c.Type.(type) {
			case *zeek.TypeSet, *zeek.TypeVector:
				for it2 := zval.Iter(val); !it2.Done(); {
					val2, _, err := it2.Next()
					if err != nil {
						return false
					}
					if bytes.Contains(val2, pattern) {
						return true
					}
				}
			default:
				if bytes.Contains(val, pattern) {
					return true
				}
			}
		}
		return false
	}
}

func combine(res expr.FieldExprResolver, pred zeek.Predicate) Filter {
	return func(r *zson.Record) bool {
		v := res(r)
		if v.Type == nil {
			// field (or sub-field) doesn't exist in this record
			return false
		}
		return pred(v)
	}
}

func CompileFieldCompare(node ast.CompareField, val zeek.Value) (Filter, error) {
	// Treat len(field) specially since we're looking at a computed
	// value rather than a field from a record.
	if op, ok := node.Field.(*ast.FieldCall); ok && op.Fn == "Len" {
		i, ok := val.(*zeek.Int)
		if !ok {
			return nil, errors.New("cannot compare len() with non-integer")
		}
		comparison, err := i.NativeComparison(node.Comparator)
		if err != nil {
			return nil, err
		}
		checklen := func(e zeek.TypedEncoding) bool {
			len, err := zeek.ContainerLength(e)
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

func EvalAny(eval zeek.Predicate, recursive bool) Filter {
	if !recursive {
		return func(r *zson.Record) bool {
			it := r.ZvalIter()
			for _, c := range r.Type.Columns {
				val, _, err := it.Next()
				if err != nil {
					return false
				}
				if eval(zeek.TypedEncoding{c.Type, val}) {
					return true
				}
			}
			return false
		}
	}

	var fn func(v zval.Encoding, cols []zeek.Column) bool
	fn = func(v zval.Encoding, cols []zeek.Column) bool {
		it := zval.Iter(v)
		for _, c := range cols {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			recType, isRecord := c.Type.(*zeek.TypeRecord)
			if isRecord && fn(val, recType.Columns) {
				return true
			} else if !isRecord && eval(zeek.TypedEncoding{c.Type, val}) {
				return true
			}
		}
		return false
	}
	return func(r *zson.Record) bool {
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
		return func(p *zson.Record) bool { return v.Value }, nil

	case *ast.SearchString:
		val := v.Value
		if val.Type != "string" {
			return nil, errors.New("SearchString value must be of type string")
		}
		return SearchString(val.Value), nil

	case *ast.CompareField:
		z, err := zeek.Parse(v.Value)
		if err != nil {
			return nil, err
		}

		if v.Comparator == "in" {
			resolver, err := expr.CompileFieldExpr(v.Field)
			if err != nil {
				return nil, err
			}
			eql, _ := z.Comparison("eql")
			comparison := zeek.Contains(eql)
			return combine(resolver, comparison), nil
		}

		return CompileFieldCompare(*v, z)

	case *ast.CompareAny:
		z, err := zeek.Parse(v.Value)
		if err != nil {
			return nil, err
		}

		if v.Comparator == "in" {
			eql, _ := z.Comparison("eql")
			comparison := zeek.Contains(eql)
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
