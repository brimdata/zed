package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/sam/expr"
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
	if slice, ok := e.RHS.(*dag.BinaryExpr); ok && slice.Op == ":" {
		return b.compileVamSlice(e.LHS, slice)
	}
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
		return vamexpr.NewCompare(b.zctx(), lhs, rhs, op)
	//case "+", "-", "*", "/", "%":
	//	return vamexpr.NewArithmetic(b.zctx(), lhs, rhs, op)
	//case "[":
	//	return vamexpr.NewIndexExpr(b.zctx(), lhs, rhs), nil
	default:
		return nil, fmt.Errorf("invalid binary operator %s", op)
	}
}

func (b *Builder) compileVamSlice(container dag.Expr, slice *dag.BinaryExpr) (vamexpr.Evaluator, error) {
	from, err := b.compileExprWithEmpty(slice.LHS)
	if err != nil {
		return nil, err
	}
	to, err := b.compileExprWithEmpty(slice.RHS)
	if err != nil {
		return nil, err
	}
	e, err := b.compileExpr(container)
	if err != nil {
		return nil, err
	}
	return vamexpr.NewSlice(b.zctx(), e, from, to), nil
}

func (b *Builder) compileVamUnary(unary dag.UnaryExpr) (vamexpr.Evaluator, error) {
	e, err := b.compileVamExpr(unary.Operand)
	if err != nil {
		return nil, err
	}
	switch unary.Op {
	case "-":
		return vamexpr.NewUnaryMinus(b.zctx(), e), nil
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
	var exprs []expr.Evaluator
	for _, e := range in {
		ev, err := b.compileVamExpr(e)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, ev)
	}
	return exprs, nil
}

func (b *Builder) compileRegexpMatch(match *dag.RegexpMatch) (vamexpr.Evaluator, error) {
	e, err := b.compileVamExpr(match.Expr)
	if err != nil {
		return nil, err
	}
	re, err := expr.CompileRegexp(match.Pattern)
	if err != nil {
		return nil, err
	}
	return vamexpr.NewRegexpMatch(re, e), nil
}

func (b *Builder) compileRegexpSearch(search *dag.RegexpSearch) (expr.Evaluator, error) {
	e, err := b.compileExpr(search.Expr)
	if err != nil {
		return nil, err
	}
	re, err := expr.CompileRegexp(search.Pattern)
	if err != nil {
		return nil, err
	}
	match := expr.NewRegexpBoolean(re)
	return expr.SearchByPredicate(expr.Contains(match), e), nil
}

func (b *Builder) compileVamRecordExpr(record *dag.RecordExpr) (vamexpr.Evaluator, error) {
	var elems []expr.RecordElem
	for _, elem := range record.Elems {
		switch elem := elem.(type) {
		case *dag.Field:
			e, err := b.compileVamExpr(elem.Value)
			if err != nil {
				return nil, err
			}
			elems = append(elems, vamexpr.RecordElem{
				Name:  elem.Name,
				Field: e,
			})
		case *dag.Spread:
			e, err := b.compileExpr(elem.Expr)
			if err != nil {
				return nil, err
			}
			elems = append(elems, vamexpr.RecordElem{Spread: e})
		}
	}
	return vamexpr.NewRecordExpr(b.zctx(), elems)
}

func (b *Builder) compileVamArrayExpr(array *dag.ArrayExpr) (vamexpr.Evaluator, error) {
	elems, err := b.compileVamVectorElems(array.Elems)
	if err != nil {
		return nil, err
	}
	return vamexpr.NewArrayExpr(b.zctx(), elems), nil
}

func (b *Builder) compileVamSetExpr(set *dag.SetExpr) (vamexpr.Evaluator, error) {
	elems, err := b.compileVamVectorElems(set.Elems)
	if err != nil {
		return nil, err
	}
	return vamexpr.NewSetExpr(b.zctx(), elems), nil
}

func (b *Builder) compileVamVectorElems(elems []dag.VectorElem) ([]vamexpr.VectorElem, error) {
	var out []vamexpr.VectorElem
	for _, elem := range elems {
		switch elem := elem.(type) {
		case *dag.Spread:
			e, err := b.compileVamExpr(elem.Expr)
			if err != nil {
				return nil, err
			}
			out = append(out, vamexpr.VectorElem{Spread: e})
		case *dag.VectorValue:
			e, err := b.compileVamExpr(elem.Expr)
			if err != nil {
				return nil, err
			}
			out = append(out, vamexpr.VectorElem{Value: e})
		}
	}
	return out, nil
}

func (b *Builder) compileVamMapExpr(m *dag.MapExpr) (vamexpr.Evaluator, error) {
	var entries []expr.Entry
	for _, f := range m.Entries {
		key, err := b.compileVamExpr(f.Key)
		if err != nil {
			return nil, err
		}
		val, err := b.compileVamExpr(f.Value)
		if err != nil {
			return nil, err
		}
		entries = append(entries, vamexpr.Entry{Key: key, Val: val})
	}
	return vamexpr.NewMapExpr(b.zctx(), entries), nil
}
