package compiler

import (
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Filter wraps an ast.BooleanExpr and implements the filter.Program interface
// so that scanners can generate filters and buffer filters from an AST without
// importing compiler (and causing an import loop).
type Filter struct {
	zctx *resolver.Context
	ast  ast.BooleanExpr
}

func NewFilter(zctx *resolver.Context, ast ast.BooleanExpr) *Filter {
	return &Filter{zctx, ast}
}

// AsFilter implements filter.Program
func (f *Filter) AsFilter() (filter.Filter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileFilter(f.zctx, f.ast)
}

// AsBufferFilter implements filter.Program
func (f *Filter) AsBufferFilter() (*filter.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileBufferFilter(f.ast)
}

func (f *Filter) AST() ast.BooleanExpr {
	return f.ast
}

func (f *Filter) AsProc() ast.Proc {
	return ast.FilterToProc(f.ast)
}

func CompileFieldCompare(zctx *resolver.Context, node *ast.CompareField) (filter.Filter, error) {
	literal := node.Value
	// Treat len(field) specially since we're looking at a computed
	// value rather than a field from a record.

	// XXX we need to implement proper expressions
	// XXX we took out field call and need to put expressions back into
	// the search sytnax, which is tricky syntactically to mix this stuff
	// with keyword search

	comparison, err := filter.Comparison(node.Comparator, literal)
	if err != nil {
		return nil, err
	}
	resolver, err := CompileExpr(zctx, node.Field)
	if err != nil {
		return nil, err
	}
	return filter.Combine(resolver, comparison), nil
}

func compileSearch(node *ast.Search) (filter.Filter, error) {
	if node.Value.Type == "regexp" || node.Value.Type == "net" {
		match, err := filter.Comparison("=~", node.Value)
		if err != nil {
			return nil, err
		}
		contains := filter.Contains(match)
		pred := func(zv zng.Value) bool {
			return match(zv) || contains(zv)
		}

		return filter.EvalAny(pred, true), nil
	}

	if node.Value.Type == "string" {
		term, err := zng.TypeBstring.Parse([]byte(node.Value.Value))
		if err != nil {
			return nil, err
		}
		return filter.SearchRecordString(string(term)), nil
	}

	return filter.SearchRecordOther(node.Text, node.Value)
}

func CompileFilter(zctx *resolver.Context, node ast.BooleanExpr) (filter.Filter, error) {
	switch v := node.(type) {
	case *ast.LogicalNot:
		expr, err := CompileFilter(zctx, v.Expr)
		if err != nil {
			return nil, err
		}
		return filter.LogicalNot(expr), nil

	case *ast.LogicalAnd:
		left, err := CompileFilter(zctx, v.Left)
		if err != nil {
			return nil, err
		}
		right, err := CompileFilter(zctx, v.Right)
		if err != nil {
			return nil, err
		}
		return filter.LogicalAnd(left, right), nil

	case *ast.LogicalOr:
		left, err := CompileFilter(zctx, v.Left)
		if err != nil {
			return nil, err
		}
		right, err := CompileFilter(zctx, v.Right)
		if err != nil {
			return nil, err
		}
		return filter.LogicalOr(left, right), nil

	case *ast.MatchAll:
		return func(*zng.Record) bool { return true }, nil

	case *ast.Search:
		return compileSearch(v)

	case *ast.CompareField:
		if v.Comparator == "in" {
			resolver, err := CompileExpr(zctx, v.Field)
			if err != nil {
				return nil, err
			}
			eql, _ := filter.Comparison("=", v.Value)
			comparison := filter.Contains(eql)
			return filter.Combine(resolver, comparison), nil
		}

		return CompileFieldCompare(zctx, v)

	case *ast.BinaryExpression:
		predicate, err := CompileExpr(zctx, v)
		if err != nil {
			return nil, err
		}
		return func(rec *zng.Record) bool {
			zv, err := predicate.Eval(rec)
			if err != nil {
				return false
			}
			if zv.Type == zng.TypeBool && zng.IsTrue(zv.Bytes) {
				return true
			}
			return false
		}, nil

	case *ast.CompareAny:
		if v.Comparator == "in" {
			compare, err := filter.Comparison("=", v.Value)
			if err != nil {
				return nil, err
			}
			contains := filter.Contains(compare)
			return filter.EvalAny(contains, v.Recursive), nil
		}
		comparison, err := filter.Comparison(v.Comparator, v.Value)
		if err != nil {
			return nil, err
		}
		return filter.EvalAny(comparison, v.Recursive), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}
