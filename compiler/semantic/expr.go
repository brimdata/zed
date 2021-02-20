package semantic

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/compiler/kernel"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/expr/agg"
	"github.com/brimsec/zq/expr/function"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng/resolver"
)

func semLiteral(literal *ast.Literal) (*kernel.ConstExpr, error) {
        return nil, nil
}

func semExpr(zctx *resolver.Context, scope *kernel.Scope, e ast.Expression) (kernel.Expr, error) {
	switch e := e.(type) {
	case *ast.Empty:
		return nil, errors.New("empty expression outside of slice")
	case *ast.Literal:
		return semLiteral(e)
	case *ast.Identifier:
		// If the identifier refers to a named variable in scope (like "$"),
		// then return a Var expression referring to the pointer to the value.
		// Note that constants may be accessed this way too by entering their
		// names into the global (outermost) scope in the Scope entity.
		if ref := scope.Lookup(e.Name); ref != nil {
                        //XXX these need to carry consts, types, functions, etc
			return &kernel.Ref{
                                Op: "Ref",
                                Name: e.Name,
                        }, nil
		}
		return rootField(e.Name), nil
	case *ast.RootRecord:
		return &kernel.Dot{"Dot"}, nil
	case *ast.UnaryExpression:
		return semUnary(zctx, scope, e)
	case *ast.SelectExpression:
		return nil, errors.New("select expression found outside of generator context")
	case *ast.BinaryExpression:
		return semBinaryExpr(zctx, scope, e)
	case *ast.ConditionalExpression:
		return semCondExpr(zctx, scope, e)
	case *ast.FunctionCall:
		return semCallExpr(zctx, scope, e)
	case *ast.CastExpression:
		return semCastExpr(zctx, scope, e)
	default:
		return nil, fmt.Errorf("invalid expression type %T", e)
	}
}

func compileExprWithEmpty(zctx *resolver.Context, scope *kernel.Scope, node ast.Expression) (expr.Evaluator, error) {
	if _, ok := node.(*ast.Empty); ok {
		return nil, nil
	}
	return compileExpr(zctx, scope, node)
}

func rootField(name string) *kernel.BinaryExpr {
	return &kernel.BinaryExpr{
		Op:       "BinaryExpr",
		Operator: ".",
		LHS:      &kernel.Dot{"Dot"},
		RHS:      &kernel.Identifier{"Op": "Identifier", Name: name},
	}
}

func CompileExprs(zctx *resolver.Context, scope *kernel.Scope, nodes []ast.Expression) ([]expr.Evaluator, error) {
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

func semBinaryExpr(zctx *resolver.Context, scope *kernel.Scope, b *ast.BinaryExpression) (*kernel.BinaryExpr, error) {
        op := b.Operator
	if op == "." {
		return semDotExpr(zctx, scope, b.LHS,b.RHS)
	}
	if op == "@" {
		return nil, errors.New("generator expression encountered outside of aggregator")
	}
	if slice, ok := RHS.(*ast.BinaryExpression); ok && slice.Operator == ":" {
		return semSlice(zctx, scope, b.LHS, slice)
	}
	lhs, err := semExpr(zctx, scope, b.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := compileExpr(zctx, scope, b.RHS)
	if err != nil {
		return nil, err
	}
        return kernel.BinarExpr{
                Op: "BinaryExpr",
                LHS: lhs,
                RHS, rhs,
        }, nil
}

func compileGenerator(zctx *resolver.Context, scope *kernel.Scope, e ast.Expression) (expr.Generator, error) {
	switch e := e.(type) {
	case *ast.BinaryExpression:
		if e.Operator != "@" {
			return nil, fmt.Errorf("bad expression in generator: %s", e.Operator)
		}
		src, err := compileGenerator(zctx, scope, e.LHS)
		if err != nil {
			return nil, err
		}
		return compileMethod(zctx, scope, src, e.RHS)
	case *ast.SelectExpression:
		exprs, err := compileExprs(zctx, scope, e.Selectors)
		if err != nil {
			return nil, err
		}
		return expr.NewSelector(exprs), nil
	}
	return nil, fmt.Errorf("bad generator expression: %T", e)
}

func compileMethod(zctx *resolver.Context, scope *kernel.Scope, src expr.Generator, e ast.Expression) (expr.Generator, error) {
	method, ok := e.(*ast.FunctionCall)
	if !ok {
		return nil, fmt.Errorf("bad method expression: %T", e)
	}
	switch method.Function {
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
		return nil, fmt.Errorf("uknown method: %s", method.Function)
	}
}

// compileMethodChain tries to compile the given function call as an aggregation
// function operating on a generator.  If this fails with an error or (nil,nil),
// then it doesn't have the required shape and the caller can try compiling
// this AST node a different way.
func compileMethodChain(zctx *resolver.Context, scope *kernel.Scope, call ast.FunctionCall) (expr.Evaluator, error) {
	if len(call.Args) != 1 {
		return nil, nil
	}
	pattern, err := agg.NewPattern(call.Function)
	if err != nil {
		return nil, nil
	}
	gen, err := compileGenerator(zctx, scope, call.Args[0])
	if gen == nil || err != nil {
		return nil, nil
	}
	return expr.NewAggExpr(zctx, pattern, gen), nil
}

func semUnary(zctx *resolver.Context, scope *kernel.Scope, u *ast.UnaryExpression) (*kernel.UnaryExpr, error) {
	if u.Operator != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", u.Operator)
	}
	e, err := semExpr(zctx, scope, u.Operand)
	if err != nil {
		return nil, err
	}
        return &kernel.Unary{
                Op: "Unary",
                Operator: p.Operator,
                Operand: e,
        }, nil
}

func semCondExpr(zctx *resolver.Context, scope *kernel.Scope, e *ast.ConditionalExpression) (*kernel.CondExpr, error) {
	predicate, err := semExpr(zctx, scope, e.Condition)
	if err != nil {
		return nil, err
	}
	thenExpr, err := semExpr(zctx, scope, e.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := semExpr(zctx, scope, e.Else)
	if err != nil {
		return nil, err
	}
	return &kernel.CondExpr{
                Op: "CondExpr",
                Condition: predicate,
                Then: thenExpr,
                Else, elseExpr,
        }, nil
}

func compileDotExpr(zctx *resolver.Context, scope *kernel.Scope, lhs, rhs ast.Expression) (expr.Evaluator, error) {
	id, ok := rhs.(*ast.Identifier)
	if !ok {
		return nil, errors.New("rhs of dot expression is not an identifier")
	}
	record, err := compileExpr(zctx, scope, lhs)
	if err != nil {
		return nil, err
	}
	return expr.NewDotAccess(record, id.Name), nil
}

func semCastExpr(zctx *resolver.Context, scope *kernel.Scope, cast *ast.CastExpression) (*kernel.CastExpr, error) {
	e, err := semExpr(zctx, scope, cast.Expr)
	if err != nil {
		return nil, err
	}
	return &kernel.CastExpr{
                Op: "CastExpr",
                Expr: e,
                Type: typ,
        }
}

func CompileLval(node ast.Expression) (field.Static, error) {
	switch node := node.(type) {
	case *ast.RootRecord:
		return field.NewRoot(), nil
	// XXX We need to allow index operators at some point, but for now
	// we have been assuming only dotted field lvalues.  See Issue #1462.
	case *ast.BinaryExpression:
		if node.Operator != "." {
			break
		}
		id, ok := node.RHS.(*ast.Identifier)
		if !ok {
			return nil, errors.New("rhs of dot operator is not an identifier")
		}
		lhs, err := CompileLval(node.LHS)
		if err != nil {
			return nil, err
		}
		return append(lhs, id.Name), nil
	case *ast.Identifier:
		return CompileLval(rootField(node.Name))
	}
	return nil, errors.New("invalid expression on lhs of assignment")
}

var ErrInference = errors.New("assignment name could not be inferred from rhs expression")

func CompileAssignment(zctx *resolver.Context, scope *kernel.Scope, node *ast.Assignment) (expr.Assignment, error) {
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
		case *ast.RootRecord:
			lhs = field.New(".")
		case *ast.Identifier:
			lhs = field.New(rhs.Name)
		case *ast.FunctionCall:
			lhs = field.New(rhs.Function)
		case *ast.BinaryExpression:
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

func compileCutter(zctx *resolver.Context, scope *kernel.Scope, node ast.FunctionCall) (*expr.Cutter, error) {
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
		return expr.Cast | expr.Crop | expr.Fill | expr.Order
	default:
		return 0
	}

}

func isShaperFunc(name string) bool {
	return shaperOps(name) != 0
}

func compileShaper(zctx *resolver.Context, scope *kernel.Scope, node ast.FunctionCall) (*expr.Shaper, error) {
	if len(node.Args) < 2 {
		return nil, function.ErrTooFewArgs
	}
	if len(node.Args) > 2 {
		return nil, function.ErrTooManyArgs
	}
	field, err := compileExpr(zctx, scope, node.Args[0])
	if err != nil {
		return nil, err
	}
	ev, err := compileExpr(zctx, scope, node.Args[1])
	if err != nil {
		return nil, err
	}
	return expr.NewShaper(zctx, field, ev, shaperOps(node.Function))
}

func semCallExpr(zctx *resolver.Context, scope *kernel.Scope, function *ast.FunctionCall) (kernel.Expr, error) {
	if e, err := semMethodChain(zctx, scope, node); e != nil || err != nil {
		return e, err
	}
	nargs := len(node.Args)
	args := node.Args
	if root {
		args = append([]ast.Expression{&ast.RootRecord{}}, args...)
	}
	exprs, err := compileExprs(zctx, scope, args)
	if err != nil {
		return nil, fmt.Errorf("%s: bad argument: %w", node.Function, err)
	}
	return expr.NewCall(zctx, fn, exprs), nil
}

func semExprs(zctx *resolver.Context, scope *kernel.Scope, in []ast.Expression) ([]kernel.Expr, error) {
	exprs := make([]kernel.Expr, 0, len(in))
	for _, e := range in {
		expr, err := semExpr(zctx, scope, e)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}
	return expr, nil
}
