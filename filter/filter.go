package filter

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/mccanne/zq/ast"
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
		checklen := func(typ zeek.Type, val []byte) bool {
			len, err := zeek.ContainerLength(typ, val)
			if err != nil {
				return false
			}
			return comparison(int64(len))
		}
		return EvalField(op.Field, checklen), nil
	}

	comparison, err := val.Comparison(node.Comparator)
	if err != nil {
		return nil, err
	}
	switch op := node.Field.(type) {
	case *ast.FieldRead:
		return EvalField(op.Field, comparison), nil
	case *ast.FieldCall:
		switch op.Fn {
		// Len handled above
		case "Index":
			idx, err := strconv.ParseInt(op.Param, 10, 64)
			if err != nil {
				return nil, err
			}
			checkidx := func(typ zeek.Type, val []byte) bool {
				elType, elVal, err := zeek.VectorIndex(typ, val, idx)
				if err != nil {
					return false
				}
				return comparison(elType, elVal)
			}
			return EvalField(op.Field, checkidx), nil
		default:
			return nil, fmt.Errorf("unknown FieldCall: %s", op.Fn)
		}
	default:
		return nil, errors.New("filter AST unknown field op")
	}
}

func EvalField(field string, eval zeek.Predicate) Filter {
	return func(p *zson.Record) bool {
		col, ok := p.Descriptor.LUT[field]
		if !ok {
			// field name doesn't exist in this record
			return false
		}
		return eval(p.TypeOfColumn(col), p.Slice(col))
	}
}

func EvalAny(eval zeek.Predicate) Filter {
	return func(p *zson.Record) bool {
		it := p.ZvalIter()
		for _, c := range p.Type.Columns {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			if eval(c.Type, val) {
				return true
			}
		}
		return false
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
			fieldRead, ok := v.Field.(*ast.FieldRead)
			if !ok {
				return nil, errors.New("type mismatch for in comparison on computed value")
			}
			eql, _ := z.Comparison("eql")
			comparison := zeek.Contains(eql)
			return EvalField(fieldRead.Field, comparison), nil
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
			return EvalAny(comparison), nil
		}

		comparison, err := z.Comparison(v.Comparator)
		if err != nil {
			return nil, err
		}
		return EvalAny(comparison), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}
