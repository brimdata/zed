package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/function"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zson"
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
func compileExpr(zctx *zed.Context, e dag.Expr) (expr.Evaluator, error) {
	if e == nil {
		return nil, errors.New("null expression not allowed")
	}
	switch e := e.(type) {
	case *dag.Literal:
		val, err := zson.ParseValue(zctx, e.Value)
		if err != nil {
			return nil, err
		}
		return expr.NewLiteral(&val), nil
	case *dag.Var:
		return expr.NewVar(e.Slot), nil
	case *dag.Search:
		return compileSearch(zctx, e)
	case *dag.This:
		return expr.NewDottedExpr(zctx, field.Path(e.Path)), nil
	case *dag.Dot:
		return compileDotExpr(zctx, e)
	case *dag.UnaryExpr:
		return compileUnary(zctx, *e)
	case *dag.BinaryExpr:
		return compileBinary(zctx, e)
	case *dag.Conditional:
		return compileConditional(zctx, *e)
	case *dag.Call:
		return compileCall(zctx, *e)
	case *dag.Cast:
		return compileCast(zctx, *e)
	case *dag.RegexpMatch:
		return compileRegexpMatch(zctx, e)
	case *dag.RegexpSearch:
		return compileFilter(zctx, e)
	case *dag.RecordExpr:
		return compileRecordExpr(zctx, e)
	case *dag.ArrayExpr:
		return compileArrayExpr(zctx, e)
	case *dag.SetExpr:
		return compileSetExpr(zctx, e)
	case *dag.MapExpr:
		return compileMapExpr(zctx, e)
	case *dag.Agg:
		agg, err := compileAgg(zctx, e)
		if err != nil {
			return nil, err
		}
		return expr.NewAggregatorExpr(agg), nil
	default:
		return nil, fmt.Errorf("invalid expression type %T", e)
	}
}

func compileExprWithEmpty(zctx *zed.Context, e dag.Expr) (expr.Evaluator, error) {
	if e == nil {
		return nil, nil
	}
	return compileExpr(zctx, e)
}

func CompileExprs(zctx *zed.Context, nodes []dag.Expr) ([]expr.Evaluator, error) {
	var exprs []expr.Evaluator
	for k := range nodes {
		e, err := compileExpr(zctx, nodes[k])
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}
	return exprs, nil
}

func compileBinary(zctx *zed.Context, e *dag.BinaryExpr) (expr.Evaluator, error) {
	if slice, ok := e.RHS.(*dag.BinaryExpr); ok && slice.Op == ":" {
		return compileSlice(zctx, e.LHS, slice)
	}
	lhs, err := compileExpr(zctx, e.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := compileExpr(zctx, e.RHS)
	if err != nil {
		return nil, err
	}
	switch op := e.Op; op {
	case "and", "or":
		return compileLogical(zctx, lhs, rhs, op)
	case "in":
		return expr.NewIn(zctx, lhs, rhs), nil
	case "=", "!=":
		return expr.NewCompareEquality(lhs, rhs, op)
	case "<", "<=", ">", ">=":
		return expr.NewCompareRelative(zctx, lhs, rhs, op)
	case "+", "-", "*", "/", "%":
		return expr.NewArithmetic(zctx, lhs, rhs, op)
	case "[":
		return expr.NewIndexExpr(zctx, lhs, rhs), nil
	default:
		return nil, fmt.Errorf("Z kernel: invalid binary operator %s", op)
	}
}

func compileSlice(zctx *zed.Context, container dag.Expr, slice *dag.BinaryExpr) (expr.Evaluator, error) {
	from, err := compileExprWithEmpty(zctx, slice.LHS)
	if err != nil {
		return nil, err
	}
	to, err := compileExprWithEmpty(zctx, slice.RHS)
	if err != nil {
		return nil, err
	}
	e, err := compileExpr(zctx, container)
	if err != nil {
		return nil, err
	}
	return expr.NewSlice(zctx, e, from, to), nil
}

func compileUnary(zctx *zed.Context, unary dag.UnaryExpr) (expr.Evaluator, error) {
	if unary.Op != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", unary.Op)
	}
	e, err := compileExpr(zctx, unary.Operand)
	if err != nil {
		return nil, err
	}
	return expr.NewLogicalNot(zctx, e), nil
}

func compileLogical(zctx *zed.Context, lhs, rhs expr.Evaluator, operator string) (expr.Evaluator, error) {
	switch operator {
	case "and":
		return expr.NewLogicalAnd(zctx, lhs, rhs), nil
	case "or":
		return expr.NewLogicalOr(zctx, lhs, rhs), nil
	default:
		return nil, fmt.Errorf("unknown logical operator: %s", operator)
	}
}

func compileConditional(zctx *zed.Context, node dag.Conditional) (expr.Evaluator, error) {
	predicate, err := compileExpr(zctx, node.Cond)
	if err != nil {
		return nil, err
	}
	thenExpr, err := compileExpr(zctx, node.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := compileExpr(zctx, node.Else)
	if err != nil {
		return nil, err
	}
	return expr.NewConditional(zctx, predicate, thenExpr, elseExpr), nil
}

func compileDotExpr(zctx *zed.Context, dot *dag.Dot) (expr.Evaluator, error) {
	record, err := compileExpr(zctx, dot.LHS)
	if err != nil {
		return nil, err
	}
	return expr.NewDotExpr(zctx, record, dot.RHS), nil
}

func compileCast(zctx *zed.Context, node dag.Cast) (expr.Evaluator, error) {
	e, err := compileExpr(zctx, node.Expr)
	if err != nil {
		return nil, err
	}
	//XXX We should handle runtime resolution of typedef names.  Issue #1572.
	typ, err := zson.ParseType(zctx, node.Type)
	if err != nil {
		return nil, err
	}
	return expr.NewCast(zctx, e, typ)
}

func compileLval(e dag.Expr) (field.Path, error) {
	if this, ok := e.(*dag.This); ok {
		return field.Path(this.Path), nil
	}
	return nil, errors.New("invalid expression on lhs of assignment")
}

func CompileAssignment(zctx *zed.Context, node *dag.Assignment) (expr.Assignment, error) {
	lhs, err := compileLval(node.LHS)
	if err != nil {
		return expr.Assignment{}, err
	}
	rhs, err := compileExpr(zctx, node.RHS)
	if err != nil {
		return expr.Assignment{}, fmt.Errorf("rhs of assigment expression: %w", err)
	}
	return expr.Assignment{lhs, rhs}, err
}

func CompileAssignments(zctx *zed.Context, dsts field.List, srcs field.List) (field.List, []expr.Evaluator) {
	if len(srcs) != len(dsts) {
		panic("CompileAssignments: argument mismatch")
	}
	var resolvers []expr.Evaluator
	var fields field.List
	for k, dst := range dsts {
		fields = append(fields, dst)
		resolvers = append(resolvers, expr.NewDottedExpr(zctx, srcs[k]))
	}
	return fields, resolvers
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

func compileShaper(zctx *zed.Context, node dag.Call) (*expr.Shaper, error) {
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
	field, err := compileExpr(zctx, args[0])
	if err != nil {
		return nil, err
	}
	typExpr, err := compileExpr(zctx, args[1])
	if err != nil {
		return nil, err
	}
	// XXX When we do constant folding, we should detect when typeExpr is
	// a constant and allocate a ConstShaper instead of a (dynamic) Shaper.
	// See issue #2425.
	return expr.NewShaper(zctx, field, typExpr, shaperOps(node.Name)), nil
}

func compileCall(zctx *zed.Context, call dag.Call) (expr.Evaluator, error) {
	// For now, we special case stateful functions here.  We should generalize this
	// as we will add many more stateful functions and also resolve this
	// the changes to create running aggegation functions from reducers.
	// XXX See issue #1259.
	switch {
	case call.Name == "missing":
		exprs, err := compileExprs(zctx, call.Args)
		if err != nil {
			return nil, fmt.Errorf("missing(): bad argument: %w", err)
		}
		return expr.NewMissing(exprs), nil
	case call.Name == "has":
		exprs, err := compileExprs(zctx, call.Args)
		if err != nil {
			return nil, fmt.Errorf("has(): bad argument: %w", err)
		}
		return expr.NewHas(exprs), nil
	case call.Name == "unflatten":
		return expr.NewUnflattener(zctx), nil
	case isShaperFunc(call.Name):
		return compileShaper(zctx, call)
	}
	nargs := len(call.Args)
	fn, path, err := function.New(zctx, call.Name, nargs)
	if err != nil {
		return nil, fmt.Errorf("%s(): %w", call.Name, err)
	}
	args := call.Args
	if path != nil {
		dagPath := &dag.This{Kind: "This", Path: path}
		args = append([]dag.Expr{dagPath}, args...)
	}
	exprs, err := compileExprs(zctx, args)
	if err != nil {
		return nil, fmt.Errorf("%s(): bad argument: %w", call.Name, err)
	}
	return expr.NewCall(zctx, fn, exprs), nil
}

func compileExprs(zctx *zed.Context, in []dag.Expr) ([]expr.Evaluator, error) {
	out := make([]expr.Evaluator, 0, len(in))
	for _, e := range in {
		ev, err := compileExpr(zctx, e)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func compileRegexpMatch(zctx *zed.Context, match *dag.RegexpMatch) (expr.Evaluator, error) {
	e, err := compileExpr(zctx, match.Expr)
	if err != nil {
		return nil, err
	}
	re, err := expr.CompileRegexp(match.Pattern)
	if err != nil {
		return nil, err
	}
	return expr.NewRegexpMatch(re, e), nil
}

func compileRecordExpr(zctx *zed.Context, record *dag.RecordExpr) (expr.Evaluator, error) {
	var names []string
	var exprs []expr.Evaluator
	for _, f := range record.Fields {
		e, err := compileExpr(zctx, f.Value)
		if err != nil {
			return nil, err
		}
		names = append(names, f.Name)
		exprs = append(exprs, e)
	}
	return expr.NewRecordExpr(zctx, names, exprs), nil
}

func compileArrayExpr(zctx *zed.Context, array *dag.ArrayExpr) (expr.Evaluator, error) {
	exprs, err := compileExprs(zctx, array.Exprs)
	if err != nil {
		return nil, err
	}
	return expr.NewArrayExpr(zctx, exprs), nil
}

func compileSetExpr(zctx *zed.Context, set *dag.SetExpr) (expr.Evaluator, error) {
	exprs, err := compileExprs(zctx, set.Exprs)
	if err != nil {
		return nil, err
	}
	return expr.NewSetExpr(zctx, exprs), nil
}

func compileMapExpr(zctx *zed.Context, m *dag.MapExpr) (expr.Evaluator, error) {
	var entries []expr.Entry
	for _, f := range m.Entries {
		key, err := compileExpr(zctx, f.Key)
		if err != nil {
			return nil, err
		}
		val, err := compileExpr(zctx, f.Value)
		if err != nil {
			return nil, err
		}
		entries = append(entries, expr.Entry{key, val})
	}
	return expr.NewMapExpr(zctx, entries), nil
}
