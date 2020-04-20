package filter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/byteconv"
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

func compileSearch(node *ast.Search) (Filter, error) {
	if node.Value.Type == "regexp" {
		match, err := Comparison("=~", node.Value)
		if err != nil {
			return nil, err
		}
		contains := Contains(match)
		pred := func(zv zng.Value) bool {
			return match(zv) || contains(zv)
		}

		return EvalAny(pred, true), nil
	}

	if node.Value.Type == "string" {
		term, err := zng.TypeBstring.Parse([]byte(node.Value.Value))
		if err != nil {
			return nil, err
		}
		return searchRecordString(string(term)), nil
	}

	return searchRecordOther(node.Text, node.Value)
}

// stringSearch is like strings.Contains() but with case-insensitive
// comparison.
func stringSearch(a, b string) bool {
	alen := len(a)
	blen := len(b)

	if blen > alen {
		return false
	}

	end := alen - blen + 1
	i := 0
	for i < end {
		if strings.EqualFold(a[i:i+blen], b) {
			return true
		}
		i++
	}
	return false
}

// searchRecordOther creates a filter that searches zng records for the
// given value, which must be of a type other than (b)string.  The filter
// matches a record that contains this value either as the value of any
// field or inside any set or array.  It also matches a record if the string
// representaton of the search value appears inside inside any string-valued
// field (or inside any element of a set or array of strings).
func searchRecordOther(searchtext string, searchval ast.Literal) (Filter, error) {
	typedCompare, err := Comparison("=", searchval)
	if err != nil {
		return nil, err
	}
	compare := func(zv zng.Value) bool {
		switch zv.Type.ID() {
		case zng.IdBstring, zng.IdString:
			s := byteconv.UnsafeString(zv.Bytes)
			return stringSearch(s, searchtext)
		default:
			return typedCompare(zv)
		}
	}
	contains := Contains(compare)

	return func(r *zng.Record) bool {
		iter := r.NewFieldIter()
		for !iter.Done() {
			_, val, err := iter.Next()
			if err != nil {
				return false
			}
			if compare(val) || contains(val) {
				return true
			}
		}
		return false
	}, nil

}

// searchRecordString handles the special case of string searching -- it
// matches both field names and values.
func searchRecordString(term string) Filter {
	search := func(zv zng.Value) bool {
		switch zv.Type.ID() {
		case zng.IdBstring, zng.IdString:
			s := byteconv.UnsafeString(zv.Bytes)
			return stringSearch(s, term)
		default:
			return false
		}
	}
	searchContainer := Contains(search)

	return func(r *zng.Record) bool {
		iter := r.NewFieldIter()
		for !iter.Done() {
			name, val, err := iter.Next()
			if err != nil {
				return false
			}
			if stringSearch(name, term) || search(val) || searchContainer(val) {
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

	case *ast.MatchAll:
		return func(*zng.Record) bool { return true }, nil

	case *ast.Search:
		return compileSearch(v)

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
		comparison, err := Comparison(v.Comparator, v.Value)
		if err != nil {
			return nil, err
		}
		return EvalAny(comparison, v.Recursive), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}
