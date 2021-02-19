package kernel

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/expr/agg"
	"github.com/brimsec/zq/expr/function"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng/resolver"
)

// TestCompileExpr provides an exported entry point for unit tests
// to compile expressions (e.g., as is used by expr/expr_test.go).
//func TestCompileExpr(zctx *resolver.Context, node ast.Expression) (expr.Evaluator, error) {
//	return compileExpr(zctx, nil, node)
//}

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
// The Evaluator return by CompileExpr produces zng.Values that are stored
// in temporary buffers and may be modified on subsequent calls to Eval.
// This is intended to minimize the garbage collection needs of the inner loop
// by not allocating memory on a per-Eval basis.  For uses like filtering and
// aggregations, where the results are immediately used, this is desirable and
// efficient but for use cases like storing the results as groupby keys, the
// resulting zng.Value should be copied (e.g., via zng.Value.Copy()).
//
// TBD: string values and net.IP address do not need to be copied because they
// are allocated by go libraries and temporary buffers are not used.  This will
// change down the road when we implement no-allocation string and IP conversion.
func compileExpr(zctx *resolver.Context, scope *Scope, e Expr) (expr.Evaluator, error) {
	if scope == nil {
		// XXX The compiler should be a struct with all of these functions
		// becoming recievers.  Then, zctx and scope can be member variables
		// of the compiler.  As compiler becomes more sophisticated, we will
		// build on this structure.  See issue #2067.
		scope = &Scope{}
	}
	switch e := e.(type) {
	case *BinaryExpr:
		return compileBinaryExpr(zctx, scope, e)
	case *CallExpr:
		return compileCallExpr(zctx, scope, e)
	case *CastExpr:
		return compileCastExpr(zctx, scope, e)
	case *CondExpr:
		return compileCondExpr(zctx, scope, e)
	case *ConstExpr:
		return expr.NewConst(e.Value)
	case *Dot:
		return &expr.RootRecord{}, nil
	case *EmptyExpr:
		return nil, errors.New("system error: empty expression encountered in compiler")
	case *Identifier:
		// If the identifier refers to a named variable in scope (like "$"),
		// then return a Var expression referring to the pointer to the value.
		// Note that constants may be accessed this way too by entering their
		// names into the global (outermost) scope in the Scope entity.
		if ref := scope.Lookup(e.Name); ref != nil {
			return expr.NewVar(ref), nil
		}
		return compileExpr(zctx, scope, rootField(e.Name))
	case *SearchExpr:
		return nil, errors.New("search TBD")
	case *SeqExpr:
		return compileSeqExpr(zctx, scope, e)
	case *UnaryExpr:
		return compileUnaryExpr(zctx, scope, e)
	default:
		return nil, fmt.Errorf("semantic checker should not have allowed this expression type: %T", e)
	}
}

func rootField(name string) *BinaryExpr {
	return &BinaryExpr{
		Op:       "BinaryExpr",
		Operator: ".",
		LHS:      &Dot{"Dot"},
		RHS:      &Identifier{Name: name},
	}
}

func CompileExprs(zctx *resolver.Context, scope *Scope, in []Expr) ([]expr.Evaluator, error) {
	var out []expr.Evaluator
	for _, e := range in {
		expr, err := compileExpr(zctx, scope, e)
		if err != nil {
			return nil, err
		}
		out = append(out, expr)
	}
	return out, nil
}

func compileBinaryExpr(zctx *resolver.Context, scope *Scope, e *BinaryExpr) (expr.Evaluator, error) {
	op := e.Operator
	if op == "." {
		return compileDotExpr(zctx, scope, e.LHS, e.RHS)
	}
	if op == "@" {
		return nil, errors.New("generator expression encountered outside of aggregator")
	}
	//XXX semantic should do this and replace BinaryExpr with Slice?
	// maybe not
	if slice, ok := e.RHS.(*BinaryExpr); ok && slice.Operator == ":" {
		return compileSlice(zctx, scope, e.LHS, slice.LHS, slice.RHS)
	}
	lhs, err := compileExpr(zctx, scope, e.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := compileExpr(zctx, scope, e.RHS)
	if err != nil {
		return nil, err
	}
	switch op {
	case "and", "or":
		return compileLogical(lhs, rhs, op)
	case "in":
		return expr.NewIn(lhs, rhs), nil
	case "=", "!=":
		return expr.NewCompareEquality(lhs, rhs, op)
	case "=~", "!~":
		return expr.NewPatternMatch(lhs, rhs, op)
	case "<", "<=", ">", ">=":
		return expr.NewCompareRelative(lhs, rhs, op)
	case "+", "-", "*", "/":
		return expr.NewArithmetic(lhs, rhs, op)
	case "[":
		return expr.NewIndexExpr(zctx, lhs, rhs)
	default:
		return nil, fmt.Errorf("invalid binary operator %s", op)
	}
}

func compileSlice(zctx *resolver.Context, scope *Scope, container, from, to Expr) (expr.Evaluator, error) {
	var sliceFrom, sliceTo expr.Evaluator
	var err error
	if from != nil {
		sliceFrom, err = compileExpr(zctx, scope, from)
		if err != nil {
			return nil, err
		}
	}
	if to != nil {
		sliceTo, err = compileExpr(zctx, scope, to)
		if err != nil {
			return nil, err
		}
	}
	e, err := compileExpr(zctx, scope, container)
	if err != nil {
		return nil, err
	}
	return expr.NewSlice(e, sliceFrom, sliceTo), nil
}

func compileSeqExpr(zctx *resolver.Context, scope *Scope, seq *SeqExpr) (expr.Evaluator, error) {
	exprs, err := compileExprs(zctx, scope, seq.Selectors)
	if err != nil {
		return nil, err
	}
	generator := expr.Generator(expr.NewSelector(exprs))
	for _, m := range seq.Methods {
		generator, err := compileMethod(zctx, scope, generator, m)
	}
	pattern, err := agg.NewPattern(seq.Name)
	if err != nil {
		return nil, err
	}
	return expr.NewAggExpr(zctx, pattern, generator), nil
}

func compileMethod(zctx *resolver.Context, scope *Scope, src expr.Generator, method Method) (expr.Generator, error) {
	switch method.Name {
	case "map":
		if len(method.Args) != 1 {
			return nil, errors.New("map() method requires one argument")
		}
		mapMethod := expr.NewMapMethod(src)
		scope.Enter()
		defer scope.Exit()
		scope.Bind("$", mapMethod.Ref())
		mapExpr, err := compileExpr(zctx, scope, method.Args[0])
		if err != nil {
			return nil, err
		}
		mapMethod.Set(mapExpr)
		return mapMethod, nil
	case "filter":
		if len(method.Args) != 1 {
			return nil, errors.New("filter() method requires one argument")
		}
		filterMethod := expr.NewFilterMethod(src)
		scope.Enter()
		defer scope.Exit()
		scope.Bind("$", filterMethod.Ref())
		filterExpr, err := compileExpr(zctx, scope, method.Args[0])
		if err != nil {
			fmt.Println("ERR", err)
			return nil, err
		}
		filterMethod.Set(filterExpr)
		return filterMethod, nil
	default:
		return nil, fmt.Errorf("uknown method: %s", method.Name)
	}
}

func compileUnaryExpr(zctx *resolver.Context, scope *Scope, u *UnaryExpr) (expr.Evaluator, error) {
	if u.Operator != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", u.Operator)
	}
	e, err := compileExpr(zctx, scope, u.Operand)
	if err != nil {
		return nil, err
	}
	return expr.NewLogicalNot(e), nil
}

func compileLogical(lhs, rhs expr.Evaluator, operator string) (expr.Evaluator, error) {
	switch operator {
	case "and":
		return expr.NewLogicalAnd(lhs, rhs), nil
	case "or":
		return expr.NewLogicalOr(lhs, rhs), nil
	default:
		return nil, fmt.Errorf("unknown logical operator: %s", operator)
	}
}

func compileCondExpr(zctx *resolver.Context, scope *Scope, cond *CondExpr) (expr.Evaluator, error) {
	predicate, err := compileExpr(zctx, scope, cond.Condition)
	if err != nil {
		return nil, err
	}
	thenExpr, err := compileExpr(zctx, scope, cond.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := compileExpr(zctx, scope, cond.Else)
	if err != nil {
		return nil, err
	}
	return expr.NewConditional(predicate, thenExpr, elseExpr), nil
}

func compileDotExpr(zctx *resolver.Context, scope *Scope, lhs, rhs Expr) (expr.Evaluator, error) {
	id, ok := rhs.(*Identifier)
	if !ok {
		// XXX semantic checker
		return nil, errors.New("rhs of dot expression is not an identifier")
	}
	e, err := compileExpr(zctx, scope, lhs)
	if err != nil {
		return nil, err
	}
	return expr.NewDotAccess(e, id.Name), nil
}

func compileCastExpr(zctx *resolver.Context, scope *Scope, cast *CastExpr) (expr.Evaluator, error) {
	e, err := compileExpr(zctx, scope, cast.Expr)
	if err != nil {
		return nil, err
	}
	return expr.NewCast(e, cast.Type)
}

func CompileLval(e Expr) (field.Static, error) {
	switch e := e.(type) {
	case *Dot:
		return field.NewRoot(), nil
	// XXX We need to allow index operators at some point, but for now
	// we have been assuming only dotted field lvalues.  See Issue #1462.
	case *BinaryExpr:
		if e.Operator != "." {
			break
		}
		id, ok := e.RHS.(*Identifier)
		if !ok {
			return nil, errors.New("rhs of dot operator is not an identifier")
		}
		lhs, err := CompileLval(e.LHS)
		if err != nil {
			return nil, err
		}
		return append(lhs, id.Name), nil
	case *Identifier:
		return CompileLval(rootField(e.Name))
	}
	return nil, errors.New("invalid expression on lhs of assignment")
}

var ErrInference = errors.New("assignment name could not be inferred from rhs expression")

func CompileAssignment(zctx *resolver.Context, scope *Scope, node Assignment) (expr.Assignment, error) {
	rhs, err := compileExpr(zctx, scope, node.RHS)
	if err != nil {
		return expr.Assignment{}, fmt.Errorf("rhs of assigment expression: %w", err)
	}
	var lhs field.Static
	if node.LHS != nil {
		lhs, err = CompileLval(node.LHS)
		if err != nil {
			return expr.Assignment{}, fmt.Errorf("lhs of assigment expression: %w", err)
		}
	} else {
		switch rhs := node.RHS.(type) {
		case *Dot:
			lhs = field.New(".")
		case *Identifier:
			lhs = field.New(rhs.Name)
		case *CallExpr:
			lhs = field.New(rhs.Name)
		case *BinaryExpr:
			// This can be a dotted record or some other expression.
			// In the latter case, it might be nice to infer a name,
			// e.g., forr "count() by a+b" we could infer "sum" for
			// the name, i,e., "count() by sum=a+b".  But for now,
			// we'll just catch this as an error.
			lhs, err = CompileLval(rhs)
			if err != nil {
				err = ErrInference
			}
		default:
			err = ErrInference
		}
	}
	return expr.Assignment{lhs, rhs}, err
}

func CompileFields(scope *Scope, fields []field.Static) []expr.Evaluator {
	var exprs []expr.Evaluator
	for _, f := range fields {
		exprs = append(exprs, expr.NewDotExpr(f))
	}
	return exprs
}

func compileCutter(zctx *resolver.Context, scope *Scope, call *CallExpr) (*expr.Cutter, error) {
	var lhs []field.Static
	var rhs []expr.Evaluator
	for _, expr := range call.Args {
		// This is a bit of a hack and could be cleaed up by re-factoring
		// CompileAssigment, but for now, we create an assigment expression
		// where the LHS and RHS are the same, so that cut(id.orig_h,_path)
		// gives a value of type {id:{orig_h:ip},{_path:string}}
		// with field names that are the same as the cut names.
		// We should allow field assignments as function arguments.
		// See issue #1772.
		assignment := Assignment{LHS: expr, RHS: expr}
		compiled, err := CompileAssignment(zctx, scope, assignment)
		if err != nil {
			return nil, err
		}
		lhs = append(lhs, compiled.LHS)
		rhs = append(rhs, compiled.RHS)
	}
	return expr.NewCutter(zctx, lhs, rhs)
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
		return expr.Cast | expr.Crop | expr.Fill | expr.Order
	default:
		return 0
	}

}

func isShaperFunc(name string) bool {
	return shaperOps(name) != 0
}

func compileShaper(zctx *resolver.Context, scope *Scope, call *CallExpr) (*expr.Shaper, error) {
	if len(call.Args) < 2 {
		return nil, function.ErrTooFewArgs
	}
	if len(call.Args) > 2 {
		return nil, function.ErrTooManyArgs
	}
	field, err := compileExpr(zctx, scope, call.Args[0])
	if err != nil {
		return nil, err
	}
	ev, err := compileExpr(zctx, scope, call.Args[1])
	if err != nil {
		return nil, err
	}
	return expr.NewShaper(zctx, field, ev, shaperOps(call.Name))
}

func compileCallExpr(zctx *resolver.Context, scope *Scope, call *CallExpr) (expr.Evaluator, error) {

	// For now, we special case cut and pick here.  We shuold generalize this
	// as we will add many more stateful functions and also resolve this
	// the changes to create running aggegation functions from reducers.
	// XXX See issue #1259.
	if call.Name == "cut" {
		cut, err := compileCutter(zctx, scope, call)
		if err != nil {
			return nil, err
		}
		cut.AllowPartialCuts()
		return cut, nil
	}
	if call.Name == "pick" {
		return compileCutter(zctx, scope, call)
	}
	if isShaperFunc(call.Name) {
		return compileShaper(zctx, scope, call)
	}
	if call.Name == "exists" {
		exprs, err := compileExprs(zctx, scope, call.Args)
		if err != nil {
			return nil, fmt.Errorf("exists: bad argument: %w", err)
		}
		return expr.NewExists(zctx, exprs), nil
	}
	nargs := len(call.Args)
	fn, root, err := function.New(zctx, call.Name, nargs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", call.Name, err)
	}
	args := call.Args
	if root {
		args = append([]Expr{&Dot{}}, args...)
	}
	exprs, err := compileExprs(zctx, scope, args)
	if err != nil {
		return nil, fmt.Errorf("%s: bad argument: %w", call.Name, err)
	}
	return expr.NewCall(zctx, fn, exprs), nil
}

func compileExprs(zctx *resolver.Context, scope *Scope, in []Expr) ([]expr.Evaluator, error) {
	out := make([]expr.Evaluator, 0, len(in))
	for _, e := range in {
		ev, err := compileExpr(zctx, scope, e)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}
