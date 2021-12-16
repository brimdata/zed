package kernel

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
	"golang.org/x/text/unicode/norm"
)

type Filter struct {
	builder  *Builder
	pushdown dag.Expr
}

var _ zbuf.Filter = (*Filter)(nil)

func (f *Filter) AsFilter() (expr.Filter, error) {
	if f == nil {
		return nil, nil
	}
	return f.builder.CompileFilter(f.pushdown)
}

func (f *Filter) AsBufferFilter() (*expr.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileBufferFilter(f.pushdown)
}

func compileCompareField(zctx *zed.Context, scope *Scope, e *dag.BinaryExpr) (expr.Filter, error) {
	if e.Op == "in" {
		literal, ok := e.LHS.(*astzed.Primitive)
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
	literal, ok := e.RHS.(*astzed.Primitive)
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

func compileSearch(node *dag.Search) (expr.Filter, error) {
	if node.Value.Type == "net" {
		match, err := expr.Comparison("=", node.Value)
		if err != nil {
			return nil, err
		}
		contains := expr.Contains(match)
		pred := func(zv *zed.Value) bool {
			return match(zv) || contains(zv)
		}

		return expr.EvalAny(pred, true), nil
	}

	if node.Value.Type == "string" {
		term := norm.NFC.Bytes(zed.UnescapeBstring([]byte(node.Value.Text)))
		return expr.SearchRecordString(string(term)), nil
	}

	return expr.SearchRecordOther(node.Text, node.Value)
}

func CompileFilter(zctx *zed.Context, scope *Scope, node dag.Expr) (expr.Filter, error) {
	switch v := node.(type) {
	case *dag.RegexpMatch:
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

	case *dag.RegexpSearch:
		re, err := expr.CompileRegexp(v.Pattern)
		if err != nil {
			return nil, err
		}
		match := expr.NewRegexpBoolean(re)
		contains := expr.Contains(match)
		pred := func(val *zed.Value) bool {
			return match(val) || contains(val)
		}
		return expr.EvalAny(pred, true), nil

	case *dag.UnaryExpr:
		if v.Op != "!" {
			return nil, fmt.Errorf("unknown unary operator: %s", v.Op)
		}
		e, err := CompileFilter(zctx, scope, v.Operand)
		if err != nil {
			return nil, err
		}
		return expr.LogicalNot(e), nil

	case *astzed.Primitive:
		// This literal translation should happen elsewhere will
		// be fixed when we add ZSON literals to Zed, e.g.,
		// dag.Literal.AsBool() etc methods.
		if v.Type != "bool" {
			return nil, fmt.Errorf("bad literal type in filter compiler: %s", v.Type)
		}
		var b bool
		switch v.Text {
		case "true":
			b = true
		case "false":
		default:
			return nil, fmt.Errorf("bad boolean value in dag.Literal: %s", v.Text)
		}
		return func(*zed.Value, *expr.Scope) bool { return b }, nil

	case *dag.Search:
		return compileSearch(v)

	case *dag.BinaryExpr:
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

	case *dag.Call:
		return compileExprPredicate(zctx, scope, v)

	default:
		return nil, fmt.Errorf("unknown filter DAG type: %T", v)
	}
}

func compileExprPredicate(zctx *zed.Context, scope *Scope, e dag.Expr) (expr.Filter, error) {
	predicate, err := compileExpr(zctx, scope, e)
	if err != nil {
		return nil, err
	}
	return func(this *zed.Value, scope *expr.Scope) bool {
		val := predicate.Eval(this, scope)
		return val.Type == zed.TypeBool && zed.IsTrue(val.Bytes)
	}, nil
}
