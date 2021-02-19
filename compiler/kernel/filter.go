package kernel

import (
	"fmt"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

func compileCompareField(zctx *resolver.Context, scope *Scope, e *BinaryExpr) (expr.Filter, error) {
	if e.Operator == "in" {
		literal, ok := e.LHS.(*ConstExpr)
		if !ok {
			return nil, nil
		}
		lval, err := compileExpr(zctx, scope, e.RHS)
		if err != nil {
			return nil, err
		}
		eql, _ := expr.Comparison("=", literal.Value)
		comparison := expr.Contains(eql)
		return expr.Combine(lval, comparison), nil
	}
	literal, ok := e.RHS.(*ConstExpr)
	if !ok {
		return nil, nil
	}
	comparison, err := expr.Comparison(e.Operator, literal.Value)
	if err != nil {
		// If this fails, return no match instead of the error and
		// let later-on code detect the error as this could be a
		// non-error situation that isn't a simple comparison.
		return nil, nil
	}
	// Check if LHS is a legit lval/field before compiling the expr.
	if _, err := CompileLval(e.LHS); err != nil {
		return nil, nil
	}
	resolver, err := compileExpr(zctx, scope, e.LHS)
	if err != nil {
		return nil, err
	}
	return expr.Combine(resolver, comparison), nil
}

//XXX contains vs match?
func compileRegexp(search *RegexpExpr) (expr.Filter, error) {
	match, err := expr.RegexpComparison("=", search.Pattern)
	if err != nil {
		return nil, err
	}
	contains := expr.Contains(match)
	pred := func(zv zng.Value) bool {
		return match(zv) || contains(zv)
	}
	return expr.EvalAny(pred, true), nil
}

func compileSearch(search *SearchExpr) (expr.Filter, error) {
	typ := zng.AliasedType(search.Value.Type)
	if typ == zng.TypeNet {
		match, err := expr.Comparison("=", search.Value)
		if err != nil {
			return nil, err
		}
		contains := expr.Contains(match)
		pred := func(zv zng.Value) bool {
			return match(zv) || contains(zv)
		}

		return expr.EvalAny(pred, true), nil
	}
	if typ == zng.TypeString {
		// Bstring?  => already parsed if in a zng value?!
		//term, err := zng.TypeBstring.Parse(z...
		//if err != nil {
		//	return nil, err
		//}
		return expr.SearchRecordString(string(search.Value.Bytes)), nil
	}

	return expr.SearchRecordOther(search.Text, search.Value)
}

func compileFilter(zctx *resolver.Context, scope *Scope, e Expr) (expr.Filter, error) {
	switch e := e.(type) {
	case *UnaryExpr:
		if e.Operator != "!" {
			return nil, fmt.Errorf("unknown unary operator: %s", e.Operator)
		}
		filter, err := compileFilter(zctx, scope, e.Operand)
		if err != nil {
			return nil, err
		}
		return expr.LogicalNot(filter), nil

	case *ConstExpr:
		// XXX should be checked by semantic...
		// we should have a boolean literal type so this
		// can't happen here
		if zng.AliasedType(e.Value.Type) != zng.TypeBool {
			return nil, fmt.Errorf("bad literal type in filter compiler: %s", e.Value.Type)
		}
		if zng.IsTrue(e.Value.Bytes) {
			return func(*zng.Record) bool { return true }, nil
		}
		return func(*zng.Record) bool { return false }, nil

	case *SearchExpr:
		return compileSearch(e)

	case *BinaryExpr:
		if e.Operator == "and" {
			left, err := compileFilter(zctx, scope, e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := compileFilter(zctx, scope, e.RHS)
			if err != nil {
				return nil, err
			}
			return expr.LogicalAnd(left, right), nil
		}
		if e.Operator == "or" {
			left, err := compileFilter(zctx, scope, e.LHS)
			if err != nil {
				return nil, err
			}
			right, err := compileFilter(zctx, scope, e.RHS)
			if err != nil {
				return nil, err
			}
			return expr.LogicalOr(left, right), nil
		}
		if f, err := compileCompareField(zctx, scope, e); f != nil || err != nil {
			return f, err
		}
		return compilExprPredicate(zctx, scope, e)

	case *CallExpr:
		return compilExprPredicate(zctx, scope, e)

	case *SeqExpr:
		if f, err := compileCompareAny(e); f != nil || err != nil {
			return f, err
		}
		return compilExprPredicate(zctx, scope, e)

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", e)
	}
}

func compilExprPredicate(zctx *resolver.Context, scope *Scope, e Expr) (expr.Filter, error) {
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

func compileCompareAny(seq *SeqExpr) (expr.Filter, error) {
	literal, op, ok := isCompareAny(seq)
	if !ok {
		return nil, nil
	}
	var pred expr.Boolean
	var err error
	if op == "in" {
		comparison, err := expr.Comparison("=", literal.Value)
		if err != nil {
			return nil, err
		}
		pred = expr.Contains(comparison)
	} else {
		pred, err = expr.Comparison(op, literal.Value)
		if err != nil {
			return nil, err
		}
	}
	return expr.EvalAny(pred, false), nil
}
