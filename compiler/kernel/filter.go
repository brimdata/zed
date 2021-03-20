package kernel

import (
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

func compileCompareField(zctx *resolver.Context, scope *Scope, e *ast.BinaryExpr) (expr.Filter, error) {
	if e.Op == "in" {
		literal, ok := e.LHS.(*ast.Primitive)
		if !ok {
			// XXX If the RHS here is a literal container or a subnet,
			// we should optimize this case.  This is part of
			// epic #2341.
			return nil, nil
		}
		// Check if RHS is a legit lval/field.
		if _, err := compileLval(e.RHS); err != nil {
			return nil, nil
		}
		resolver, err := compileExpr(zctx, scope, e.RHS)
		if err != nil {
			return nil, err
		}
		eql, _ := expr.Comparison("=", *literal)
		comparison := expr.Contains(eql)
		return expr.Apply(resolver, comparison), nil
	}
	literal, ok := e.RHS.(*ast.Primitive)
	if !ok {
		return nil, nil
	}
	comparison, err := expr.Comparison(e.Op, *literal)
	if err != nil {
		// If this fails, return no match instead of the error and
		// let later-on code detect the error as this could be a
		// non-error situation that isn't a simple comparison.
		return nil, nil
	}
	// Check if LHS is a legit lval/field before compiling the expr.
	if _, err := compileLval(e.LHS); err != nil {
		return nil, nil
	}
	resolver, err := compileExpr(zctx, scope, e.LHS)
	if err != nil {
		return nil, err
	}
	return expr.Apply(resolver, comparison), nil
}

func compileSearch(node *ast.Search) (expr.Filter, error) {
	if node.Value.Type == "regexp" || node.Value.Type == "net" {
		match, err := expr.Comparison("=", node.Value)
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
		term, err := tzngio.ParseBstring([]byte(node.Value.Text))
		if err != nil {
			return nil, err
		}
		return expr.SearchRecordString(string(term)), nil
	}

	return expr.SearchRecordOther(node.Text, node.Value)
}

func CompileFilter(zctx *resolver.Context, scope *Scope, node ast.Expr) (expr.Filter, error) {
	switch v := node.(type) {
	case *ast.Regexp:
		e, err := compileExpr(zctx, scope, v.Expr)
		if err != nil {
			return nil, err
		}
		re, err := expr.CompileRegexp(v.Pattern)
		if err != nil {
			return nil, err
		}
		pred := expr.NewRegexpBoolean(re)
		return expr.Apply(e, pred), nil

	case *ast.UnaryExpr:
		if v.Op != "!" {
			return nil, fmt.Errorf("unknown unary operator: %s", v.Op)
		}
		e, err := CompileFilter(zctx, scope, v.Operand)
		if err != nil {
			return nil, err
		}
		return expr.LogicalNot(e), nil

	case *ast.Primitive:
		// This literal translation should happen elsewhere will
		// be fixed when we add ZSON literals to Z, e.g.,
		// ast.Literal.AsBool() etc methods.
		if v.Type != "bool" {
			return nil, fmt.Errorf("bad literal type in filter compiler: %s", v.Type)
		}
		var b bool
		switch v.Text {
		case "true":
			b = true
		case "false":
		default:
			return nil, fmt.Errorf("bad boolean value in ast.Literal: %s", v.Text)
		}
		return func(*zng.Record) bool { return b }, nil

	case *ast.Search:
		return compileSearch(v)

	case *ast.BinaryExpr:
		if v.Op == "and" {
			left, err := CompileFilter(zctx, scope, v.LHS)
			if err != nil {
				return nil, err
			}
			right, err := CompileFilter(zctx, scope, v.RHS)
			if err != nil {
				return nil, err
			}
			return expr.LogicalAnd(left, right), nil
		}
		if v.Op == "or" {
			left, err := CompileFilter(zctx, scope, v.LHS)
			if err != nil {
				return nil, err
			}
			right, err := CompileFilter(zctx, scope, v.RHS)
			if err != nil {
				return nil, err
			}
			return expr.LogicalOr(left, right), nil
		}
		if f, err := compileCompareField(zctx, scope, v); f != nil || err != nil {
			return f, err
		}
		return compileExprPredicate(zctx, scope, v)

	case *ast.Call:
		return compileExprPredicate(zctx, scope, v)
	case *ast.SeqExpr:
		if f, err := compileCompareAny(v); f != nil || err != nil {
			return f, err
		}
		return compileExprPredicate(zctx, scope, v)

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}

func compileExprPredicate(zctx *resolver.Context, scope *Scope, e ast.Expr) (expr.Filter, error) {
	predicate, err := compileExpr(zctx, scope, e)
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
}

func compileCompareAny(e *ast.SeqExpr) (expr.Filter, error) {
	literal, op, ok := isCompareAny(e)
	if !ok {
		return nil, nil
	}
	var pred expr.Boolean
	var err error
	if op == "in" {
		comparison, err := expr.Comparison("=", *literal)
		if err != nil {
			return nil, err
		}
		pred = expr.Contains(comparison)
	} else {
		pred, err = expr.Comparison(op, *literal)
		if err != nil {
			return nil, err
		}
	}
	return expr.EvalAny(pred, false), nil
}
