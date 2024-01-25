package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/op/combine"
	"github.com/brimdata/zed/runtime/sam/op/traverse"
	vamexpr "github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/zbuf"
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
	if e.Op == "in" {
		// Do a faster comparison if the LHS is a compile-time constant expression.
		if in, err := b.compileConstIn(e); in != nil && err == nil {
			return in, err
		}
	}
	if e, err := b.compileVamConstCompare(e); e != nil && err == nil {
		return e, nil
	}
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
	case "in":
		return vamexpr.NewIn(b.zctx(), lhs, rhs), nil
	case "==", "!=":
		return vamexpr.NewCompareEquality(b.zctx(), lhs, rhs, op)
	case "<", "<=", ">", ">=":
		return vamexpr.NewCompareRelative(b.zctx(), lhs, rhs, op)
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

func (b *Builder) compileShaper(node dag.Call, tf expr.ShaperTransform) (vamexpr.Evaluator, error) {
	args := node.Args
	field, err := b.compileExpr(args[0])
	if err != nil {
		return nil, err
	}
	typExpr, err := b.compileExpr(args[1])
	if err != nil {
		return nil, err
	}
	return expr.NewShaper(b.zctx(), field, typExpr, tf)
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

func (b *Builder) compileRegexpMatch(match *dag.RegexpMatch) (expr.Evaluator, error) {
	e, err := b.compileExpr(match.Expr)
	if err != nil {
		return nil, err
	}
	re, err := expr.CompileRegexp(match.Pattern)
	if err != nil {
		return nil, err
	}
	return expr.NewRegexpMatch(re, e), nil
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

func (b *Builder) compileRecordExpr(record *dag.RecordExpr) (expr.Evaluator, error) {
	var elems []expr.RecordElem
	for _, elem := range record.Elems {
		switch elem := elem.(type) {
		case *dag.Field:
			e, err := b.compileExpr(elem.Value)
			if err != nil {
				return nil, err
			}
			elems = append(elems, expr.RecordElem{
				Name:  elem.Name,
				Field: e,
			})
		case *dag.Spread:
			e, err := b.compileExpr(elem.Expr)
			if err != nil {
				return nil, err
			}
			elems = append(elems, expr.RecordElem{Spread: e})
		}
	}
	return expr.NewRecordExpr(b.zctx(), elems)
}

func (b *Builder) compileArrayExpr(array *dag.ArrayExpr) (expr.Evaluator, error) {
	elems, err := b.compileVectorElems(array.Elems)
	if err != nil {
		return nil, err
	}
	return expr.NewArrayExpr(b.zctx(), elems), nil
}

func (b *Builder) compileSetExpr(set *dag.SetExpr) (expr.Evaluator, error) {
	elems, err := b.compileVectorElems(set.Elems)
	if err != nil {
		return nil, err
	}
	return expr.NewSetExpr(b.zctx(), elems), nil
}

func (b *Builder) compileVectorElems(elems []dag.VectorElem) ([]expr.VectorElem, error) {
	var out []expr.VectorElem
	for _, elem := range elems {
		switch elem := elem.(type) {
		case *dag.Spread:
			e, err := b.compileExpr(elem.Expr)
			if err != nil {
				return nil, err
			}
			out = append(out, expr.VectorElem{Spread: e})
		case *dag.VectorValue:
			e, err := b.compileExpr(elem.Expr)
			if err != nil {
				return nil, err
			}
			out = append(out, expr.VectorElem{Value: e})
		}
	}
	return out, nil
}

func (b *Builder) compileVamMapExpr(m *dag.MapExpr) (expr.Evaluator, error) {
	var entries []expr.Entry
	for _, f := range m.Entries {
		key, err := b.compileVamExpr(f.Key)
		if err != nil {
			return nil, err
		}
		val, err := b.compileExpr(f.Value)
		if err != nil {
			return nil, err
		}
		entries = append(entries, expr.Entry{Key: key, Val: val})
	}
	return expr.NewMapExpr(b.zctx(), entries), nil
}

func (b *Builder) compileOverExpr(over *dag.OverExpr) (expr.Evaluator, error) {
	if over.Body == nil {
		return nil, errors.New("over expression requires a lateral query body")
	}
	names, lets, err := b.compileDefs(over.Defs)
	if err != nil {
		return nil, err
	}
	exprs, err := b.compileExprs(over.Exprs)
	if err != nil {
		return nil, err
	}
	parent := traverse.NewExpr(b.rctx.Context, b.zctx())
	enter := traverse.NewOver(b.rctx, parent, exprs)
	scope := enter.AddScope(b.rctx.Context, names, lets)
	exits, err := b.compileSeq(over.Body, []zbuf.Puller{scope})
	if err != nil {
		return nil, err
	}
	var exit zbuf.Puller
	if len(exits) == 1 {
		exit = exits[0]
	} else {
		// This can happen when output of over body
		// is a fork or switch.
		exit = combine.New(b.rctx, exits)
	}
	parent.SetExit(scope.NewExit(exit))
	return parent, nil
}
