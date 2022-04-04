package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/function"
	"github.com/brimdata/zed/zson"
	"golang.org/x/text/unicode/norm"
)

// compileExpr compiles the given Expression into an object
// that evaluates the expression against a provided Record.  It returns an
// error if compilation fails for any reason.
//
// This is the "intepreted slow path" of the analytics engine.  Because it
// handles dynamic typing at runtime, overheads are incurred due to
// various type checks and coercions that determine different computational
// outcomes based on type.  There is nothing here that optimizes analytics
// for native machine types; these optimizations (will) happen in the pushdown
// predicate processing engine in the zst columnar scanner.
//
// Eventually, we will optimize this zst "fast path" by dynamically
// generating byte codes (which can in turn be JIT assembled into machine code)
// for each zng TypeRecord encountered.  Once you know the TypeRecord,
// you can generate code using strong typing just as an OLAP system does
// due to its schemas defined up-front in its relational tables.  Here,
// each record type is like a schema and as we encounter them, we can compile
// optimized code for the now-static types within that record type.
//
// The Evaluator return by CompileExpr produces zed.Values that are stored
// in temporary buffers and may be modified on subsequent calls to Eval.
// This is intended to minimize the garbage collection needs of the inner loop
// by not allocating memory on a per-Eval basis.  For uses like filtering and
// aggregations, where the results are immediately used, this is desirable and
// efficient but for use cases like storing the results as groupby keys, the
// resulting zed.Value should be copied (e.g., via zed.Value.Copy()).
//
// TBD: string values and net.IP address do not need to be copied because they
// are allocated by go libraries and temporary buffers are not used.  This will
// change down the road when we implement no-allocation string and IP conversion.
func (b *Builder) compileExpr(e dag.Expr) (expr.Evaluator, error) {
	if e == nil {
		return nil, errors.New("null expression not allowed")
	}
	switch e := e.(type) {
	case *dag.Literal:
		val, err := zson.ParseValue(b.zctx(), e.Value)
		if err != nil {
			return nil, err
		}
		return expr.NewLiteral(val), nil
	case *dag.Var:
		return expr.NewVar(e.Slot), nil
	case *dag.Search:
		return b.compileSearch(e)
	case *dag.This:
		return expr.NewDottedExpr(b.zctx(), field.Path(e.Path)), nil
	case *dag.Dot:
		return b.compileDotExpr(e)
	case *dag.UnaryExpr:
		return b.compileUnary(*e)
	case *dag.BinaryExpr:
		return b.compileBinary(e)
	case *dag.Conditional:
		return b.compileConditional(*e)
	case *dag.Call:
		return b.compileCall(*e)
	case *dag.RegexpMatch:
		return b.compileRegexpMatch(e)
	case *dag.RegexpSearch:
		return b.compileRegexpSearch(e)
	case *dag.RecordExpr:
		return b.compileRecordExpr(e)
	case *dag.ArrayExpr:
		return b.compileArrayExpr(e)
	case *dag.SetExpr:
		return b.compileSetExpr(e)
	case *dag.MapExpr:
		return b.compileMapExpr(e)
	case *dag.Agg:
		agg, err := b.compileAgg(e)
		if err != nil {
			return nil, err
		}
		return expr.NewAggregatorExpr(agg), nil
	default:
		return nil, fmt.Errorf("invalid expression type %T", e)
	}
}

func (b *Builder) compileExprWithEmpty(e dag.Expr) (expr.Evaluator, error) {
	if e == nil {
		return nil, nil
	}
	return b.compileExpr(e)
}

func (b *Builder) compileBinary(e *dag.BinaryExpr) (expr.Evaluator, error) {
	if slice, ok := e.RHS.(*dag.BinaryExpr); ok && slice.Op == ":" {
		return b.compileSlice(e.LHS, slice)
	}
	if e.Op == "in" {
		// Do a faster comparison if the LHS is a compile-time constant expression.
		if in, err := b.compileConstIn(e); in != nil && err == nil {
			return in, err
		}
	}
	if e, err := b.compileConstCompare(e); e != nil && err == nil {
		return e, nil
	}
	lhs, err := b.compileExpr(e.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := b.compileExpr(e.RHS)
	if err != nil {
		return nil, err
	}
	switch op := e.Op; op {
	case "and":
		return expr.NewLogicalAnd(b.zctx(), lhs, rhs), nil
	case "or":
		return expr.NewLogicalOr(b.zctx(), lhs, rhs), nil
	case "in":
		return expr.NewIn(b.zctx(), lhs, rhs), nil
	case "==", "!=":
		return expr.NewCompareEquality(lhs, rhs, op)
	case "<", "<=", ">", ">=":
		return expr.NewCompareRelative(b.zctx(), lhs, rhs, op)
	case "+", "-", "*", "/", "%":
		return expr.NewArithmetic(b.zctx(), lhs, rhs, op)
	case "[":
		return expr.NewIndexExpr(b.zctx(), lhs, rhs), nil
	default:
		return nil, fmt.Errorf("invalid binary operator %s", op)
	}
}

func (b *Builder) compileConstIn(e *dag.BinaryExpr) (expr.Evaluator, error) {
	literal, err := b.evalAtCompileTime(e.LHS)
	if err != nil || literal.IsError() {
		// If the RHS here is a literal value, it would be good
		// to optimize this too.  However, this will all be handled
		// by the JIT compiler that will create optimized closures
		// on a per-type basis.
		return nil, nil
	}
	eql, err := expr.Comparison("==", literal)
	if eql == nil || err != nil {
		return nil, nil
	}
	operand, err := b.compileExpr(e.RHS)
	if err != nil {
		return nil, err
	}
	return expr.NewFilter(operand, expr.Contains(eql)), nil
}

func (b *Builder) compileConstCompare(e *dag.BinaryExpr) (expr.Evaluator, error) {
	switch e.Op {
	case "==", "!=", "<", "<=", ">", ">=":
	default:
		return nil, nil
	}
	literal, err := b.evalAtCompileTime(e.RHS)
	if err != nil || literal.IsError() {
		return nil, nil
	}
	comparison, err := expr.Comparison(e.Op, literal)
	if comparison == nil || err != nil {
		// If this fails, return no match instead of the error and
		// let later-on code detect the error as this could be a
		// non-error situation that isn't a simple comparison.
		return nil, nil
	}
	operand, err := b.compileExpr(e.LHS)
	if err != nil {
		return nil, err
	}
	return expr.NewFilter(operand, comparison), nil
}

func (b *Builder) compileSearch(search *dag.Search) (expr.Evaluator, error) {
	val, err := zson.ParseValue(b.zctx(), search.Value)
	if err != nil {
		return nil, err
	}
	e, err := b.compileExpr(search.Expr)
	if err != nil {
		return nil, err
	}
	if zed.TypeUnder(val.Type) == zed.TypeString {
		// Do a grep-style substring search instead of an
		// exact match on each value.
		term := norm.NFC.Bytes(val.Bytes)
		return expr.NewSearchString(string(term), e), nil
	}
	return expr.NewSearch(search.Text, val, e)
}

func (b *Builder) compileSlice(container dag.Expr, slice *dag.BinaryExpr) (expr.Evaluator, error) {
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
	return expr.NewSlice(b.zctx(), e, from, to), nil
}

func (b *Builder) compileUnary(unary dag.UnaryExpr) (expr.Evaluator, error) {
	e, err := b.compileExpr(unary.Operand)
	if err != nil {
		return nil, err
	}
	switch unary.Op {
	case "-":
		return expr.NewUnaryMinus(b.zctx(), e), nil
	case "!":
		return expr.NewLogicalNot(b.zctx(), e), nil
	default:
		return nil, fmt.Errorf("unknown unary operator %s\n", unary.Op)
	}
}

func (b *Builder) compileConditional(node dag.Conditional) (expr.Evaluator, error) {
	predicate, err := b.compileExpr(node.Cond)
	if err != nil {
		return nil, err
	}
	thenExpr, err := b.compileExpr(node.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := b.compileExpr(node.Else)
	if err != nil {
		return nil, err
	}
	return expr.NewConditional(b.zctx(), predicate, thenExpr, elseExpr), nil
}

func (b *Builder) compileDotExpr(dot *dag.Dot) (expr.Evaluator, error) {
	record, err := b.compileExpr(dot.LHS)
	if err != nil {
		return nil, err
	}
	return expr.NewDotExpr(b.zctx(), record, dot.RHS), nil
}

func compileLval(e dag.Expr) (field.Path, error) {
	if this, ok := e.(*dag.This); ok {
		return field.Path(this.Path), nil
	}
	return nil, errors.New("invalid expression on lhs of assignment")
}

func (b *Builder) compileAssignment(node *dag.Assignment) (expr.Assignment, error) {
	lhs, err := compileLval(node.LHS)
	if err != nil {
		return expr.Assignment{}, err
	}
	rhs, err := b.compileExpr(node.RHS)
	if err != nil {
		return expr.Assignment{}, fmt.Errorf("rhs of assigment expression: %w", err)
	}
	return expr.Assignment{lhs, rhs}, err
}

func shaperOps(name string) expr.ShaperTransform {
	switch name {
	case "cast":
		return expr.Cast
	case "crop":
		return expr.Crop
	case "fill":
		return expr.Fill
	case "fit":
		return expr.Crop | expr.Fill
	case "order":
		return expr.Order
	case "shape":
		return expr.Cast | expr.Fill | expr.Order
	default:
		return 0
	}

}

func isShaperFunc(name string) bool {
	return shaperOps(name) != 0
}

func (b *Builder) compileShaper(node dag.Call) (*expr.Shaper, error) {
	args := node.Args
	if len(args) == 1 {
		args = append([]dag.Expr{&dag.This{Kind: "This"}}, args...)
	}
	if len(args) < 2 {
		return nil, function.ErrTooFewArgs
	}
	if len(args) > 2 {
		return nil, function.ErrTooManyArgs
	}
	field, err := b.compileExpr(args[0])
	if err != nil {
		return nil, err
	}
	typExpr, err := b.compileExpr(args[1])
	if err != nil {
		return nil, err
	}
	// XXX When we do constant folding, we should detect when typeExpr is
	// a constant and allocate a ConstShaper instead of a (dynamic) Shaper.
	// See issue #2425.
	return expr.NewShaper(b.zctx(), field, typExpr, shaperOps(node.Name)), nil
}

func (b *Builder) compileCall(call dag.Call) (expr.Evaluator, error) {
	// For now, we special case stateful functions here.  We should generalize this
	// as we will add many more stateful functions and also resolve this
	// the changes to create running aggegation functions from reducers.
	// XXX See issue #1259.
	switch {
	case call.Name == "missing":
		exprs, err := b.compileExprs(call.Args)
		if err != nil {
			return nil, fmt.Errorf("missing(): bad argument: %w", err)
		}
		return expr.NewMissing(exprs), nil
	case call.Name == "has":
		exprs, err := b.compileExprs(call.Args)
		if err != nil {
			return nil, fmt.Errorf("has(): bad argument: %w", err)
		}
		return expr.NewHas(exprs), nil
	case call.Name == "nest_dotted":
		return expr.NewUnflattener(b.zctx()), nil
	case isShaperFunc(call.Name):
		return b.compileShaper(call)
	}
	nargs := len(call.Args)
	fn, path, err := function.New(b.zctx(), call.Name, nargs)
	if err != nil {
		return nil, fmt.Errorf("%s(): %w", call.Name, err)
	}
	args := call.Args
	if path != nil {
		dagPath := &dag.This{Kind: "This", Path: path}
		args = append([]dag.Expr{dagPath}, args...)
	}
	exprs, err := b.compileExprs(args)
	if err != nil {
		return nil, fmt.Errorf("%s(): bad argument: %w", call.Name, err)
	}
	return expr.NewCall(b.zctx(), fn, exprs), nil
}

func (b *Builder) compileExprs(in []dag.Expr) ([]expr.Evaluator, error) {
	var exprs []expr.Evaluator
	for _, e := range in {
		ev, err := b.compileExpr(e)
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
	exprs, err := b.compileExprs(array.Exprs)
	if err != nil {
		return nil, err
	}
	return expr.NewArrayExpr(b.zctx(), exprs), nil
}

func (b *Builder) compileSetExpr(set *dag.SetExpr) (expr.Evaluator, error) {
	exprs, err := b.compileExprs(set.Exprs)
	if err != nil {
		return nil, err
	}
	return expr.NewSetExpr(b.zctx(), exprs), nil
}

func (b *Builder) compileMapExpr(m *dag.MapExpr) (expr.Evaluator, error) {
	var entries []expr.Entry
	for _, f := range m.Entries {
		key, err := b.compileExpr(f.Key)
		if err != nil {
			return nil, err
		}
		val, err := b.compileExpr(f.Value)
		if err != nil {
			return nil, err
		}
		entries = append(entries, expr.Entry{key, val})
	}
	return expr.NewMapExpr(b.zctx(), entries), nil
}
