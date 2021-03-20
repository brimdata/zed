package kernel

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/expr/agg"
	"github.com/brimsec/zq/expr/function"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zson"
)

var Root = &ast.Path{Kind: "Path", Name: []string{}}

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
func compileExpr(zctx *resolver.Context, scope *Scope, e ast.Expr) (expr.Evaluator, error) {
	if e == nil {
		return nil, errors.New("null expression not allowed")
	}
	switch e := e.(type) {
	case *ast.Primitive:
		return expr.NewLiteral(*e)
	case *ast.Id:
		// This should be converted in the semantic pass but it can come
		// over the network from a worker, so we check again.
		return nil, fmt.Errorf("Z kernel compiler: encountered AST identifier for '%s'", e.Name)
	case *ast.Root:
		return nil, fmt.Errorf("Z kernel compiler: encountered AST root record")
	case *ast.Ref:
		// If the reference refers to a named variable in scope (like "$"),
		// then return a Var expression referring to the pointer to the value.
		// Note that constants may be accessed this way too by entering their
		// names into the global (outermost) scope in the Scope entity.
		if ref := scope.Lookup(e.Name); ref != nil {
			return expr.NewVar(ref), nil
		}
		return nil, fmt.Errorf("unknown reference: '%s'", e.Name)
	case *ast.Path:
		return expr.NewDotExpr(field.Static(e.Name)), nil
	case *ast.UnaryExpr:
		return compileUnary(zctx, scope, *e)
	case *ast.SelectExpr:
		return nil, errors.New("Z kernel: encountered select expression")
	case *ast.BinaryExpr:
		return compileBinary(zctx, scope, e)
	case *ast.Conditional:
		return compileConditional(zctx, scope, *e)
	case *ast.Call:
		return compileCall(zctx, scope, *e)
	case *ast.Cast:
		return compileCast(zctx, scope, *e)
	case *ast.TypeValue:
		return compileTypeValue(zctx, scope, e)
	case *ast.SeqExpr:
		return compileSeqExpr(zctx, scope, e)
	case *ast.RegexpMatch:
		return compileRegexpMatch(zctx, scope, e)
	default:
		return nil, fmt.Errorf("invalid expression type %T", e)
	}
}

func compileExprWithEmpty(zctx *resolver.Context, scope *Scope, e ast.Expr) (expr.Evaluator, error) {
	if e == nil {
		return nil, nil
	}
	return compileExpr(zctx, scope, e)
}

func CompileExprs(zctx *resolver.Context, scope *Scope, nodes []ast.Expr) ([]expr.Evaluator, error) {
	var exprs []expr.Evaluator
	for k := range nodes {
		e, err := compileExpr(zctx, scope, nodes[k])
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}
	return exprs, nil
}

func compileBinary(zctx *resolver.Context, scope *Scope, e *ast.BinaryExpr) (expr.Evaluator, error) {
	if slice, ok := e.RHS.(*ast.BinaryExpr); ok && slice.Op == ":" {
		return compileSlice(zctx, scope, e.LHS, slice)
	}
	lhs, err := compileExpr(zctx, scope, e.LHS)
	if err != nil {
		return nil, err
	}
	if e.Op == "." {
		// We should change this to DotExpr.  See issue #2255.
		id, ok := e.RHS.(*ast.Id)
		if !ok {
			return nil, fmt.Errorf("Z kernel: RHS of dot operator is not a name")
		}
		return expr.NewDotAccess(lhs, id.Name), nil
	}
	rhs, err := compileExpr(zctx, scope, e.RHS)
	if err != nil {
		return nil, err
	}
	switch op := e.Op; op {
	case "and", "or":
		return compileLogical(lhs, rhs, op)
	case "in":
		return expr.NewIn(lhs, rhs), nil
	case "=", "!=":
		return expr.NewCompareEquality(lhs, rhs, op)
	case "<", "<=", ">", ">=":
		return expr.NewCompareRelative(lhs, rhs, op)
	case "+", "-", "*", "/":
		return expr.NewArithmetic(lhs, rhs, op)
	case "[":
		return expr.NewIndexExpr(zctx, lhs, rhs)
	default:
		return nil, fmt.Errorf("Z kernel: invalid binary operator %s", op)
	}
}

func compileSlice(zctx *resolver.Context, scope *Scope, container ast.Expr, slice *ast.BinaryExpr) (expr.Evaluator, error) {
	from, err := compileExprWithEmpty(zctx, scope, slice.LHS)
	if err != nil {
		return nil, err
	}
	to, err := compileExprWithEmpty(zctx, scope, slice.RHS)
	if err != nil {
		return nil, err
	}
	e, err := compileExpr(zctx, scope, container)
	if err != nil {
		return nil, err
	}
	return expr.NewSlice(e, from, to), nil
}

func compileSeqExpr(zctx *resolver.Context, scope *Scope, seq *ast.SeqExpr) (expr.Evaluator, error) {
	selectors, err := compileExprs(zctx, scope, seq.Selectors)
	if err != nil {
		return nil, err
	}
	selector := expr.NewSelector(selectors)
	sequence := expr.Generator(selector)
	for _, method := range seq.Methods {
		sequence, err = compileMethod(zctx, scope, sequence, method)
		if err != nil {
			return nil, err
		}
	}
	pattern, err := agg.NewPattern(seq.Name)
	if err != nil {
		return nil, err
	}
	return expr.NewAggExpr(zctx, pattern, sequence), nil
}

func compileMethod(zctx *resolver.Context, scope *Scope, src expr.Generator, method ast.Method) (expr.Generator, error) {
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

func compileUnary(zctx *resolver.Context, scope *Scope, unary ast.UnaryExpr) (expr.Evaluator, error) {
	if unary.Op != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", unary.Op)
	}
	e, err := compileExpr(zctx, scope, unary.Operand)
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

func compileConditional(zctx *resolver.Context, scope *Scope, node ast.Conditional) (expr.Evaluator, error) {
	predicate, err := compileExpr(zctx, scope, node.Cond)
	if err != nil {
		return nil, err
	}
	thenExpr, err := compileExpr(zctx, scope, node.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := compileExpr(zctx, scope, node.Else)
	if err != nil {
		return nil, err
	}
	return expr.NewConditional(predicate, thenExpr, elseExpr), nil
}

func compileDotExpr(zctx *resolver.Context, scope *Scope, lhs, rhs ast.Expr) (expr.Evaluator, error) {
	id, ok := rhs.(*ast.Id)
	if !ok {
		return nil, errors.New("rhs of dot expression is not an identifier")
	}
	record, err := compileExpr(zctx, scope, lhs)
	if err != nil {
		return nil, err
	}
	return expr.NewDotAccess(record, id.Name), nil
}

func compileCast(zctx *resolver.Context, scope *Scope, node ast.Cast) (expr.Evaluator, error) {
	e, err := compileExpr(zctx, scope, node.Expr)
	if err != nil {
		return nil, err
	}
	//XXX we should handle runtime resolution of typedef names
	typ, err := zson.TranslateType(zctx.Context, node.Type)
	if err != nil {
		return nil, err
	}
	return expr.NewCast(e, typ)
}

func compileLval(e ast.Expr) (field.Static, error) {
	if e, ok := e.(*ast.Path); ok {
		return field.Static(e.Name), nil
	}
	return nil, errors.New("invalid expression on lhs of assignment")
}

func CompileAssignment(zctx *resolver.Context, scope *Scope, node *ast.Assignment) (expr.Assignment, error) {
	lhs, err := compileLval(node.LHS)
	if err != nil {
		return expr.Assignment{}, err
	}
	rhs, err := compileExpr(zctx, scope, node.RHS)
	if err != nil {
		return expr.Assignment{}, fmt.Errorf("rhs of assigment expression: %w", err)
	}
	return expr.Assignment{lhs, rhs}, err
}

func CompileAssignments(dsts []field.Static, srcs []field.Static) ([]field.Static, []expr.Evaluator) {
	if len(srcs) != len(dsts) {
		panic("CompileAssignments: argument mismatch")
	}
	var resolvers []expr.Evaluator
	var fields []field.Static
	for k, dst := range dsts {
		fields = append(fields, dst)
		resolvers = append(resolvers, expr.NewDotExpr(srcs[k]))
	}
	return fields, resolvers
}

func compileCutter(zctx *resolver.Context, scope *Scope, node ast.Call) (*expr.Cutter, error) {
	var lhs []field.Static
	var rhs []expr.Evaluator
	for _, expr := range node.Args {
		// This is a bit of a hack and could be cleaed up by re-factoring
		// CompileAssigment, but for now, we create an assigment expression
		// where the LHS and RHS are the same, so that cut(id.orig_h,_path)
		// gives a value of type {id:{orig_h:ip},{_path:string}}
		// with field names that are the same as the cut names.
		// We should allow field assignments as function arguments.
		// See issue #1772.
		assignment := &ast.Assignment{LHS: expr, RHS: expr}
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
		return expr.Cast | expr.Fill | expr.Order
	default:
		return 0
	}

}

func isShaperFunc(name string) bool {
	return shaperOps(name) != 0
}

func compileShaper(zctx *resolver.Context, scope *Scope, node ast.Call) (*expr.Shaper, error) {
	args := node.Args
	if len(args) == 1 {
		args = append([]ast.Expr{Root}, args...)
	}
	if len(args) < 2 {
		return nil, function.ErrTooFewArgs
	}
	if len(args) > 2 {
		return nil, function.ErrTooManyArgs
	}
	field, err := compileExpr(zctx, scope, args[0])
	if err != nil {
		return nil, err
	}
	ev, err := compileExpr(zctx, scope, args[1])
	if err != nil {
		return nil, err
	}
	return expr.NewShaper(zctx, field, ev, shaperOps(node.Name))
}

func compileCall(zctx *resolver.Context, scope *Scope, call ast.Call) (expr.Evaluator, error) {
	// For now, we special case stateful functions here.  We shuold generalize this
	// as we will add many more stateful functions and also resolve this
	// the changes to create running aggegation functions from reducers.
	// XXX See issue #1259.
	switch {
	case call.Name == "cut":
		cut, err := compileCutter(zctx, scope, call)
		if err != nil {
			return nil, err
		}
		cut.AllowPartialCuts()
		return cut, nil
	case call.Name == "pick":
		return compileCutter(zctx, scope, call)
	case call.Name == "exists":
		exprs, err := compileExprs(zctx, scope, call.Args)
		if err != nil {
			return nil, fmt.Errorf("exists: bad argument: %w", err)
		}
		return expr.NewExists(zctx, exprs), nil
	case call.Name == "unflatten":
		return expr.NewUnflattener(zctx), nil
	case isShaperFunc(call.Name):
		return compileShaper(zctx, scope, call)
	}
	nargs := len(call.Args)
	fn, root, err := function.New(zctx.Context, call.Name, nargs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", call.Name, err)
	}
	args := call.Args
	if root {
		args = append([]ast.Expr{Root}, args...)
	}
	exprs, err := compileExprs(zctx, scope, args)
	if err != nil {
		return nil, fmt.Errorf("%s: bad argument: %w", call.Name, err)
	}
	return expr.NewCall(zctx, fn, exprs), nil
}

func compileExprs(zctx *resolver.Context, scope *Scope, in []ast.Expr) ([]expr.Evaluator, error) {
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

func compileTypeValue(zctx *resolver.Context, scope *Scope, t *ast.TypeValue) (expr.Evaluator, error) {
	if typ, ok := t.Value.(*ast.TypeName); ok {
		// We currently support dynamic type names only for
		// top-level type names.  By dynamic, we mean typedefs that
		// come from the data instead of the Z.  For dynamic type
		// names that are embedded lower down in a complex type,
		// we need to implement some type of tracker objec that
		// can resolve the type when all the dependent types are found.
		// See issue #2182.
		return expr.NewTypeFunc(zctx, typ.Name), nil
	}
	typ, err := zson.TranslateType(zctx.Context, t.Value)
	if err != nil {
		return nil, err
	}
	return expr.NewLiteralVal(zng.NewTypeType(typ)), nil
}

func compileRegexpMatch(zctx *resolver.Context, scope *Scope, match *ast.RegexpMatch) (expr.Evaluator, error) {
	e, err := compileExpr(zctx, scope, match.Expr)
	if err != nil {
		return nil, err
	}
	re, err := expr.CompileRegexp(match.Pattern)
	if err != nil {
		return nil, err
	}
	return expr.NewRegexpMatch(re, e), nil
}

func compileRegexpSearch(zctx *resolver.Context, scope *Scope, search *ast.RegexpSearch) (expr.Evaluator, error) {
	re, err := expr.CompileRegexp(search.Pattern)
	if err != nil {
		return nil, err
	}
	return expr.NewRegexpSearch(re), nil
}
