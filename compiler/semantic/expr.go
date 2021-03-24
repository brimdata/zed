package semantic

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/expr/agg"
)

func semExpr(scope *Scope, e ast.Expr) (ast.Expr, error) {
	switch e := e.(type) {
	case nil:
		return nil, errors.New("semantic analysis: illegal null value encountered in AST")
	case *ast.RegexpMatch:
		if _, err := expr.CompileRegexp(e.Pattern); err != nil {
			return nil, err
		}
		converted, err := semExpr(scope, e.Expr)
		if err != nil {
			return nil, err
		}
		return &ast.RegexpMatch{
			Kind:    "RegexpMatch",
			Pattern: e.Pattern,
			Expr:    converted,
		}, nil
	case *ast.RegexpSearch:
		return &ast.RegexpSearch{
			Kind:    "RegexpSearch",
			Pattern: e.Pattern,
		}, nil
	case *ast.Primitive:
		return e, nil
	case *ast.Id:
		// We use static scoping here to see if an identifier is
		// a "var" reference to the name or a field access
		// and transform the ast node appropriately.  The semantic AST
		// should have no Identifiers in it as they should all be
		// resolved one way or another.
		if ref := scope.Lookup(e.Name); ref != nil {
			// For now, this could only be a literal but
			// it may refer to other data types down the
			// road so we call it a "ref" for now.
			return &ast.Ref{"Ref", e.Name}, nil
		}
		if e.Name == "$" {
			return &ast.Ref{"Ref", "$"}, nil
		}
		return semField(scope, e)
	case *ast.Root:
		return semField(scope, e)
	case *ast.Search:
		return e, nil
	case *ast.UnaryExpr:
		expr, err := semExpr(scope, e.Operand)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{
			Kind:    "UnaryExpr",
			Op:      e.Op,
			Operand: expr,
		}, nil
	case *ast.SelectExpr:
		return nil, errors.New("select expression found outside of generator context")
	case *ast.BinaryExpr:
		return semBinary(scope, e)
	case *ast.Conditional:
		cond, err := semExpr(scope, e.Cond)
		if err != nil {
			return nil, err
		}
		thenExpr, err := semExpr(scope, e.Then)
		if err != nil {
			return nil, err
		}
		elseExpr, err := semExpr(scope, e.Else)
		if err != nil {
			return nil, err
		}
		return &ast.Conditional{
			Kind: "Conditional",
			Cond: cond,
			Then: thenExpr,
			Else: elseExpr,
		}, nil
	case *ast.Call:
		return semCall(scope, e)
	case *ast.Cast:
		expr, err := semExpr(scope, e.Expr)
		if err != nil {
			return nil, err
		}
		return &ast.Cast{
			Kind: "Cast",
			Expr: expr,
			Type: e.Type, //XXX
		}, nil
	case *ast.TypeValue:
		tv, err := semType(scope, e.Value)
		if err != nil {
			return nil, err
		}
		return &ast.TypeValue{
			Kind:  "TypeValue",
			Value: tv,
		}, nil
	case *ast.Agg:
		expr, err := semExprNullable(scope, e.Expr)
		if err != nil {
			return nil, err
		}
		if expr == nil && e.Name != "count" {
			return nil, fmt.Errorf("aggregator '%s' requires argument", e.Name)
		}
		where, err := semExprNullable(scope, e.Where)
		if err != nil {
			return nil, err
		}
		return &ast.Agg{
			Kind:  "Agg",
			Name:  e.Name,
			Expr:  expr,
			Where: where,
		}, nil
	}
	return nil, fmt.Errorf("invalid expression type %T", e)
}

func semBinary(scope *Scope, e *ast.BinaryExpr) (ast.Expr, error) {
	op := e.Op
	if op == "." {
		return semField(scope, e)
	}
	if slice, ok := e.RHS.(*ast.BinaryExpr); ok && slice.Op == ":" {
		if op != "[" {
			return nil, errors.New("slice outside of index operator")
		}
		ref, err := semExpr(scope, e.LHS)
		if err != nil {
			return nil, err
		}
		slice, err := semSlice(scope, slice)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpr{
			Kind: "BinaryExpr",
			Op:   "[",
			LHS:  ref,
			RHS:  slice,
		}, nil
	}
	lhs, err := semExpr(scope, e.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := semExpr(scope, e.RHS)
	if err != nil {
		return nil, err
	}
	// If we index a path expression with a string constant, then just
	// extend the path...
	if op == "[" {
		if path := isPathIndex(scope, lhs, rhs); path != nil {
			return path, nil
		}
	}
	return &ast.BinaryExpr{
		Kind: "BinaryExpr",
		Op:   e.Op,
		LHS:  lhs,
		RHS:  rhs,
	}, nil
}

func isPathIndex(scope *Scope, lhs, rhs ast.Expr) *ast.Path {
	if path, ok := lhs.(*ast.Path); ok {
		if s, ok := isStringConst(scope, rhs); ok {
			path.Name = append(path.Name, s)
			return path
		}
	}
	return nil
}

func isStringConst(scope *Scope, e ast.Expr) (string, bool) {
	if p, ok := e.(*ast.Primitive); ok && p.Type == "string" {
		return p.Text, true
	}
	if ref, ok := e.(*ast.Ref); ok {
		if p := scope.Lookup(ref.Name); p != nil {
			if c, ok := p.(*ast.Const); ok {
				return isStringConst(scope, c.Expr)
			}
		}
	}
	return "", false
}

func semSlice(scope *Scope, slice *ast.BinaryExpr) (*ast.BinaryExpr, error) {
	sliceFrom, err := semExprNullable(scope, slice.LHS)
	if err != nil {
		return nil, err
	}
	sliceTo, err := semExprNullable(scope, slice.RHS)
	if err != nil {
		return nil, err
	}
	return &ast.BinaryExpr{
		Kind: "BinaryExpr",
		Op:   ":",
		LHS:  sliceFrom,
		RHS:  sliceTo,
	}, nil
}

func semExprNullable(scope *Scope, e ast.Expr) (ast.Expr, error) {
	if e == nil {
		return nil, nil
	}
	return semExpr(scope, e)
}

func semCall(scope *Scope, call *ast.Call) (ast.Expr, error) {
	if e, err := semSequence(scope, call); e != nil || err != nil {
		return e, err
	}
	exprs, err := semExprs(scope, call.Args)
	if err != nil {
		return nil, fmt.Errorf("%s: bad argument: %w", call.Name, err)
	}
	return &ast.Call{
		Kind: "Call",
		Name: call.Name,
		Args: exprs,
	}, nil
}

func semSequence(scope *Scope, call *ast.Call) (*ast.SeqExpr, error) {
	if len(call.Args) != 1 {
		return nil, nil
	}
	sel, ok := call.Args[0].(*ast.SelectExpr)
	if !ok {
		return nil, nil
	}
	_, err := agg.NewPattern(call.Name)
	if err != nil {
		return nil, nil
	}
	selectors, err := semExprs(scope, sel.Selectors)
	if err != nil {
		return nil, nil
	}
	var methods []ast.Method
	for _, call := range sel.Methods {
		m, err := semMethod(scope, call)
		if err != nil {
			return nil, err
		}
		methods = append(methods, *m)
	}
	return &ast.SeqExpr{
		Kind:      "SeqExpr",
		Name:      call.Name,
		Selectors: selectors,
		Methods:   methods,
	}, nil
}

func semMethod(scope *Scope, call ast.Call) (*ast.Method, error) {
	switch call.Name {
	case "map":
		if len(call.Args) != 1 {
			return nil, errors.New("map() method requires one argument")
		}
		scope.Enter()
		defer scope.Exit()
		e, err := semExpr(scope, call.Args[0])
		scope.Bind("$", nil)
		if err != nil {
			return nil, err
		}
		return &ast.Method{Name: "map", Args: []ast.Expr{e}}, nil
	case "filter":
		if len(call.Args) != 1 {
			return nil, errors.New("filter() method requires one argument")
		}
		scope.Enter()
		defer scope.Exit()
		scope.Bind("$", nil)
		e, err := semExpr(scope, call.Args[0])
		if err != nil {
			return nil, err
		}
		return &ast.Method{Name: "filter", Args: []ast.Expr{e}}, nil
	default:
		return nil, fmt.Errorf("uknown method: %s", call.Name)
	}
}

func semExprs(scope *Scope, in []ast.Expr) ([]ast.Expr, error) {
	exprs := make([]ast.Expr, 0, len(in))
	for _, e := range in {
		expr, err := semExpr(scope, e)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}
	return exprs, nil
}

func semAssignments(scope *Scope, assignments []ast.Assignment) ([]ast.Assignment, error) {
	out := make([]ast.Assignment, 0, len(assignments))
	for _, e := range assignments {
		a, err := semAssignment(scope, e)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func semAssignment(scope *Scope, a ast.Assignment) (ast.Assignment, error) {
	rhs, err := semExpr(scope, a.RHS)
	if err != nil {
		return ast.Assignment{}, fmt.Errorf("rhs of assigment expression: %w", err)
	}
	var lhs ast.Expr
	if a.LHS != nil {
		// XXX currently only support explicit field lvals
		// (i.e., no assignments to array elements etc... instead
		// you create a new array with modified contends)
		lhs, err = semField(scope, a.LHS)
		if err != nil {
			return ast.Assignment{}, fmt.Errorf("lhs of assigment expression: %w", err)
		}
	} else if call, ok := a.RHS.(*ast.Call); ok {
		lhs = &ast.Path{"Path", []string{call.Name}}
	} else if agg, ok := a.RHS.(*ast.Agg); ok {
		lhs = &ast.Path{"Path", []string{agg.Name}}
	} else if _, ok := a.RHS.(*ast.Root); ok {
		lhs = &ast.Path{"Path", []string{"."}}
	} else {
		lhs, err = semField(scope, a.RHS)
		if err != nil {
			return ast.Assignment{}, errors.New("assignment name could not be inferred from rhs expression")
		}
	}
	return ast.Assignment{"Assignment", lhs, rhs}, nil
}

func semFields(scope *Scope, exprs []ast.Expr) ([]ast.Expr, error) {
	fields := make([]ast.Expr, 0, len(exprs))
	for _, e := range exprs {
		f, err := semField(scope, e)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	return fields, nil
}

// semField checks that an expression is a field refernce and converts it
// to a field path if possible.  It will convert any references to Refs.
func semField(scope *Scope, e ast.Expr) (ast.Expr, error) {
	switch e := e.(type) {
	case *ast.BinaryExpr:
		if e.Op == "." {
			lhs, err := semField(scope, e.LHS)
			if err != nil {
				return nil, err
			}
			id, ok := e.RHS.(*ast.Id)
			if !ok {
				return nil, errors.New("RHS of dot operator is not an identifier")
			}
			if lhs, ok := lhs.(*ast.Path); ok {
				lhs.Name = append(lhs.Name, id.Name)
				return lhs, nil
			}
			return &ast.BinaryExpr{
				Kind: "BinaryExpr",
				Op:   ".",
				LHS:  lhs,
				RHS:  id,
			}, nil
		}
		if e.Op == "[" {
			lhs, err := semField(scope, e.LHS)
			if err != nil {
				return nil, err
			}
			rhs, err := semExpr(scope, e.RHS)
			if err != nil {
				return nil, err
			}
			if path := isPathIndex(scope, lhs, rhs); path != nil {
				return path, nil
			}
			return &ast.BinaryExpr{
				Kind: "BinaryExpr",
				Op:   "[",
				LHS:  lhs,
				RHS:  rhs,
			}, nil
		}
	case *ast.Id:
		if scope.Lookup(e.Name) != nil {
			// For now, this could only be a literal but
			// it may refer to other data types down the
			// road so we call it a "ref" for now.
			return &ast.Ref{"Ref", e.Name}, nil
		}
		if e.Name == "$" {
			return &ast.Ref{"Ref", "$"}, nil
		}
		return &ast.Path{Kind: "Path", Name: []string{e.Name}}, nil
	case *ast.Root:
		return &ast.Path{Kind: "Path", Name: []string{}}, nil
	}
	// This includes a null Expr, which can happen if the AST is missing
	// a field or sets it to null.
	return nil, errors.New("expression is not a field reference.")
}

// convertFunctionProc converts a FunctionCall ast node at proc level
// to a group-by or a filter-proc based on the name of the function.
// This way, Z of the form `... | exists(...) | ...` can be distinguished
// from `count()` by the name lookup here at compile time.
func convertFunctionProc(call *ast.Call) (ast.Proc, error) {
	if _, err := agg.NewPattern(call.Name); err != nil {
		// Assume it's a valid function and convert.  If not,
		// the compiler will report an unknown function error.
		return ast.FilterToProc(call), nil
	}
	var e ast.Expr
	if len(call.Args) > 1 {
		return nil, fmt.Errorf("%s: wrong number of arguments", call.Name)
	}
	if len(call.Args) == 1 {
		e = call.Args[0]
	}
	agg := &ast.Agg{
		Kind: "Agg",
		Name: call.Name,
		Expr: e,
	}
	return &ast.Summarize{
		Kind: "Summarize",
		Aggs: []ast.Assignment{
			{
				Kind: "Assignment",
				RHS:  agg,
			},
		},
	}, nil
}
