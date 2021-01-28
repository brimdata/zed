package compiler

import (
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var _ zbuf.Filter = (*Filter)(nil)

// Filter wraps an ast.BooleanExpr and implements the zbuf.Filter interface
// so that scanners can generate filters and buffer filters from an AST without
// importing compiler (and causing an import loop).
type Filter struct {
	zctx *resolver.Context
	ast  ast.BooleanExpr
}

func NewFilter(zctx *resolver.Context, ast ast.BooleanExpr) *Filter {
	return &Filter{zctx, ast}
}

func (f *Filter) AsFilter() (expr.Filter, error) {
	if f == nil {
		return nil, nil
	}
	return compileFilter(f.zctx, f.ast)
}

func (f *Filter) AsBufferFilter() (*expr.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return compileBufferFilter(f.ast)
}

func (f *Filter) AST() ast.BooleanExpr {
	return f.ast
}

func (f *Filter) AsProc() ast.Proc {
	return ast.FilterToProc(f.ast)
}

func compileFieldCompare(zctx *resolver.Context, node *ast.CompareField) (expr.Filter, error) {
	literal := node.Value
	// Treat len(field) specially since we're looking at a computed
	// value rather than a field from a record.

	// XXX we need to implement proper expressions
	// XXX we took out field call and need to put expressions back into
	// the search sytnax, which is tricky syntactically to mix this stuff
	// with keyword search

	comparison, err := expr.Comparison(node.Comparator, literal)
	if err != nil {
		return nil, err
	}
	resolver, err := CompileExpr(zctx, node.Field)
	if err != nil {
		return nil, err
	}
	return expr.Combine(resolver, comparison), nil
}

func compileSearch(node *ast.Search) (expr.Filter, error) {
	if node.Value.Type == "regexp" || node.Value.Type == "net" {
		match, err := expr.Comparison("=~", node.Value)
		if err != nil {
			return nil, err
		}
		contains := expr.Contains(match)
		pred := func(zv zng.Value) bool {
			return match(zv) || contains(zv)
		}

		return expr.EvalAny(pred, true), nil
	}

	if node.Value.Type == "string" {
		term, err := zng.TypeBstring.Parse([]byte(node.Value.Value))
		if err != nil {
			return nil, err
		}
		return expr.SearchRecordString(string(term)), nil
	}

	return expr.SearchRecordOther(node.Text, node.Value)
}

func compileFilter(zctx *resolver.Context, node ast.BooleanExpr) (expr.Filter, error) {
	switch v := node.(type) {
	case *ast.LogicalNot:
		e, err := compileFilter(zctx, v.Expr)
		if err != nil {
			return nil, err
		}
		return expr.LogicalNot(e), nil

	case *ast.LogicalAnd:
		left, err := compileFilter(zctx, v.Left)
		if err != nil {
			return nil, err
		}
		right, err := compileFilter(zctx, v.Right)
		if err != nil {
			return nil, err
		}
		return expr.LogicalAnd(left, right), nil

	case *ast.LogicalOr:
		left, err := compileFilter(zctx, v.Left)
		if err != nil {
			return nil, err
		}
		right, err := compileFilter(zctx, v.Right)
		if err != nil {
			return nil, err
		}
		return expr.LogicalOr(left, right), nil

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
			eql, _ := expr.Comparison("=", v.Value)
			comparison := expr.Contains(eql)
			return expr.Combine(resolver, comparison), nil
		}

		return compileFieldCompare(zctx, v)

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
			compare, err := expr.Comparison("=", v.Value)
			if err != nil {
				return nil, err
			}
			contains := expr.Contains(compare)
			return expr.EvalAny(contains, v.Recursive), nil
		}
		comparison, err := expr.Comparison(v.Comparator, v.Value)
		if err != nil {
			return nil, err
		}
		return expr.EvalAny(comparison, v.Recursive), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}
