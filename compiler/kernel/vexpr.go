package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
	vamexpr "github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/zson"
)

func (b *Builder) compileVamExpr(e dag.Expr) (vamexpr.Evaluator, error) {
	if e == nil {
		return nil, errors.New("null expression not allowed")
	}
	switch e := e.(type) {
	case *dag.Literal:
		val, err := zson.ParseValue(b.zctx(), e.Value)
		if err != nil {
			return nil, err
		}
		return vamexpr.NewLiteral(val), nil
	//case *dag.Var:
	//	return vamexpr.NewVar(e.Slot), nil
	//case *dag.Search:
	//	return b.compileSearch(e)
	case *dag.This:
		return vamexpr.NewDottedExpr(b.zctx(), field.Path(e.Path)), nil
	case *dag.Dot:
		return b.compileVamDotExpr(e)
	case *dag.UnaryExpr:
		return b.compileVamUnary(*e)
	case *dag.BinaryExpr:
		return b.compileVamBinary(e)
	//case *dag.Conditional:
	//	return b.compileVamConditional(*e)
	//case *dag.Call:
	//	return b.compileVamCall(*e)
	//case *dag.RegexpMatch:
	//	return b.compileVamRegexpMatch(e)
	//case *dag.RegexpSearch:
	//	return b.compileVamRegexpSearch(e)
	//case *dag.RecordExpr:
	//	return b.compileVamRecordExpr(e)
	//case *dag.ArrayExpr:
	//	return b.compileVamArrayExpr(e)
	//case *dag.SetExpr:
	//	return b.compileVamSetExpr(e)
	//case *dag.MapCall:
	//	return b.compileVamMapCall(e)
	//case *dag.MapExpr:
	//	return b.compileVamMapExpr(e)
	//case *dag.Agg:
	//	agg, err := b.compileAgg(e)
	//	if err != nil {
	//		return nil, err
	//	}
	//	return expr.NewAggregatorExpr(agg), nil
	//case *dag.OverExpr:
	//	return b.compileOverExpr(e)
	default:
		return nil, fmt.Errorf("vector expression type %T: not supported", e)
	}
}

func (b *Builder) compileVamExprWithEmpty(e dag.Expr) (vamexpr.Evaluator, error) {
	if e == nil {
		return nil, nil
	}
	return b.compileVamExpr(e)
}

func (b *Builder) compileVamBinary(e *dag.BinaryExpr) (vamexpr.Evaluator, error) {
	//XXX TBD
	//if slice, ok := e.RHS.(*dag.BinaryExpr); ok && slice.Op == ":" {
	//	return b.compileVamSlice(e.LHS, slice)
	//}

	//XXX TBD
	//if e.Op == "in" {
	// Do a faster comparison if the LHS is a compile-time constant expression.
	//	if in, err := b.compileConstIn(e); in != nil && err == nil {
	//		return in, err
	//	}
	//}
	// XXX don't think we need this... callee can check for const
	//if e, err := b.compileVamConstCompare(e); e != nil && err == nil {
	//	return e, nil
	//}
	lhs, err := b.compileVamExpr(e.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := b.compileVamExpr(e.RHS)
	if err != nil {
		return nil, err
	}
	switch op := e.Op; op {
	case "and":
		return vamexpr.NewLogicalAnd(b.zctx(), lhs, rhs), nil
	case "or":
		return vamexpr.NewLogicalOr(b.zctx(), lhs, rhs), nil
	//case "in": XXX TBD
	//	return vamexpr.NewIn(b.zctx(), lhs, rhs), nil
	case "==", "!=", "<", "<=", ">", ">=":
		return vamexpr.NewCompare(b.zctx(), lhs, rhs, op), nil
	//case "+", "-", "*", "/", "%":
	//	return vamexpr.NewArithmetic(b.zctx(), lhs, rhs, op)
	//case "[":
	//	return vamexpr.NewIndexExpr(b.zctx(), lhs, rhs), nil
	default:
		return nil, fmt.Errorf("invalid binary operator %s", op)
	}
}

func (b *Builder) compileVamUnary(unary dag.UnaryExpr) (vamexpr.Evaluator, error) {
	e, err := b.compileVamExpr(unary.Operand)
	if err != nil {
		return nil, err
	}
	switch unary.Op {
	//XXX TBD
	//case "-":
	//	return vamexpr.NewUnaryMinus(b.zctx(), e), nil
	case "!":
		return vamexpr.NewLogicalNot(b.zctx(), e), nil
	default:
		return nil, fmt.Errorf("unknown unary operator %s", unary.Op)
	}
}

func (b *Builder) compileVamDotExpr(dot *dag.Dot) (vamexpr.Evaluator, error) {
	record, err := b.compileVamExpr(dot.LHS)
	if err != nil {
		return nil, err
	}
	return vamexpr.NewDotExpr(b.zctx(), record, dot.RHS), nil
}

func (b *Builder) compileVamExprs(in []dag.Expr) ([]vamexpr.Evaluator, error) {
	var exprs []vamexpr.Evaluator
	for _, e := range in {
		ev, err := b.compileVamExpr(e)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, ev)
	}
	return exprs, nil
}
