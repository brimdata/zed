package kernel

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"golang.org/x/text/unicode/norm"
)

type Filter struct {
	builder  *Builder
	pushdown dag.Expr
}

var _ zbuf.Filter = (*Filter)(nil)

func (f *Filter) AsEvaluator() (expr.Evaluator, error) {
	if f == nil {
		return nil, nil
	}
	b := f.builder
	return compileFilter(b.pctx.Zctx, f.pushdown)
}

func (f *Filter) AsBufferFilter() (*expr.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileBufferFilter(f.builder.pctx.Zctx, f.pushdown)
}

func compileCompareField(zctx *zed.Context, e *dag.BinaryExpr) (expr.Evaluator, error) {
	if e.Op == "in" {
		literal, err := isLiteral(zctx, e.LHS)
		if err != nil {
			return nil, err
		}
		if literal == nil {
			// XXX If the RHS here is a literal container or a subnet,
			// we should optimize this case.  This is part of
			// epic #2341.
			return nil, nil
		}
		// Check if RHS is a legit lval/field.
		if _, err := compileLval(e.RHS); err != nil {
			return nil, nil
		}
		field, err := compileExpr(zctx, e.RHS)
		if err != nil {
			return nil, err
		}
		eql, _ := expr.Comparison("=", literal)
		predicate := expr.Contains(eql)
		return expr.NewFilter(field, predicate), nil
	}
	literal, err := isLiteral(zctx, e.RHS)
	if literal == nil {
		return nil, err
	}
	comparison, err := expr.Comparison(e.Op, literal)
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
	field, err := compileExpr(zctx, e.LHS)
	if err != nil {
		return nil, err
	}
	return expr.NewFilter(field, comparison), nil
}

func isLiteral(zctx *zed.Context, e dag.Expr) (*zed.Value, error) {
	if literal, ok := e.(*dag.Literal); ok {
		val, err := zson.ParseValue(zctx, literal.Value)
		if err != nil {
			return nil, err
		}
		return &val, nil
	}
	return nil, nil
}

func compileSearch(zctx *zed.Context, search *dag.Search) (expr.Evaluator, error) {
	val, err := zson.ParseValue(zctx, search.Value)
	if err != nil {
		return nil, err
	}
	switch zed.TypeUnder(val.Type) {
	case zed.TypeNet:
		match, err := expr.Comparison("=", &val)
		if err != nil {
			return nil, err
		}
		contains := expr.Contains(match)
		pred := func(val *zed.Value) bool {
			return match(val) || contains(val)
		}
		return expr.SearchByPredicate(pred), nil
	case zed.TypeString:
		term := norm.NFC.Bytes(val.Bytes)
		return expr.NewSearchString(string(term)), nil
	}
	return expr.NewSearch(search.Text, &val)
}

func compileFilter(zctx *zed.Context, node dag.Expr) (expr.Evaluator, error) {
	switch v := node.(type) {
	case *dag.RegexpMatch:
		e, err := compileExpr(zctx, v.Expr)
		if err != nil {
			return nil, err
		}
		re, err := expr.CompileRegexp(v.Pattern)
		if err != nil {
			return nil, err
		}
		pred := expr.NewRegexpBoolean(re)
		return expr.NewFilter(e, pred), nil

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
		return expr.SearchByPredicate(pred), nil

	case *dag.UnaryExpr:
		if v.Op != "!" {
			return nil, fmt.Errorf("unknown unary operator: %s", v.Op)
		}
		e, err := compileFilter(zctx, v.Operand)
		if err != nil {
			return nil, err
		}
		return expr.NewLogicalNot(zctx, e), nil

	case *dag.Search:
		return compileSearch(zctx, v)

	case *dag.BinaryExpr:
		if v.Op == "and" {
			left, err := compileFilter(zctx, v.LHS)
			if err != nil {
				return nil, err
			}
			right, err := compileFilter(zctx, v.RHS)
			if err != nil {
				return nil, err
			}
			return expr.NewLogicalAnd(zctx, left, right), nil
		}
		if v.Op == "or" {
			left, err := compileFilter(zctx, v.LHS)
			if err != nil {
				return nil, err
			}
			right, err := compileFilter(zctx, v.RHS)
			if err != nil {
				return nil, err
			}
			return expr.NewLogicalOr(zctx, left, right), nil
		}
		if f, err := compileCompareField(zctx, v); f != nil || err != nil {
			return f, err
		}
		return compileExpr(zctx, v)

	case *dag.Call, *dag.Literal:
		return compileExpr(zctx, v)

	default:
		return nil, fmt.Errorf("unknown filter DAG type: %T", v)
	}
}
