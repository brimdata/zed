package semantic

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/agg"
	"github.com/brimdata/zed/expr/function"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zson"
)

func semExpr(scope *Scope, e ast.Expr) (dag.Expr, error) {
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
		return &dag.RegexpMatch{
			Kind:    "RegexpMatch",
			Pattern: e.Pattern,
			Expr:    converted,
		}, nil
	case *ast.RegexpSearch:
		return &dag.RegexpSearch{
			Kind:    "RegexpSearch",
			Pattern: e.Pattern,
		}, nil
	case *astzed.Primitive:
		val, err := zson.ParsePrimitive(e.Type, e.Text)
		if err != nil {
			return nil, err
		}
		return &dag.Literal{
			Kind:  "Literal",
			Value: zson.MustFormatValue(val),
		}, nil
	case *ast.ID:
		return semID(scope, e), nil
	case *ast.Search:
		val, err := zson.ParsePrimitive(e.Value.Type, e.Value.Text)
		if err != nil {
			return nil, err
		}
		return &dag.Search{
			Kind:  "Search",
			Text:  e.Text,
			Value: zson.MustFormatValue(val),
		}, nil
	case *ast.UnaryExpr:
		expr, err := semExpr(scope, e.Operand)
		if err != nil {
			return nil, err
		}
		return &dag.UnaryExpr{
			Kind:    "UnaryExpr",
			Op:      e.Op,
			Operand: expr,
		}, nil
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
		return &dag.Conditional{
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
		typ, err := semType(scope, e.Type)
		if err != nil {
			return nil, err
		}
		return &dag.Cast{
			Kind: "Cast",
			Expr: expr,
			Type: typ,
		}, nil
	case *astzed.TypeValue:
		typ, err := semType(scope, e.Value)
		if err != nil {
			// If this is a type name, then we check to see if it's in the
			// context because it has been defined locally.  If not, then
			// the type needs to come from the data, in which case we replace
			// the literal reference with a typename() call.
			// Note that we just check the top value here but there can be
			// nested dynamic type references inside a complex type; this
			// is not yet supported and will fail here with a compile-time error
			// complaining about the type not existing.
			// XXX See issue #3413
			if e := semDynamicType(scope, e.Value); e != nil {
				return e, nil
			}
			return nil, err
		}
		return &dag.Literal{
			Kind:  "Literal",
			Value: "<" + typ + ">",
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
		return &dag.Agg{
			Kind:  "Agg",
			Name:  e.Name,
			Expr:  expr,
			Where: where,
		}, nil
	case *ast.RecordExpr:
		var out []dag.RecordElem
		for _, elem := range e.Elems {
			switch elem := elem.(type) {
			case *ast.Field:
				e, err := semExpr(scope, elem.Value)
				if err != nil {
					return nil, err
				}
				out = append(out, &dag.Field{
					Kind:  "Field",
					Name:  elem.Name,
					Value: e,
				})
			case *ast.ID:
				out = append(out, &dag.Field{
					Kind:  "Field",
					Name:  elem.Name,
					Value: semID(scope, elem),
				})
			case *ast.Spread:
				e, err := semExpr(scope, elem.Expr)
				if err != nil {
					return nil, err
				}
				out = append(out, &dag.Spread{
					Kind: "Spread",
					Expr: e,
				})
			}
		}
		return &dag.RecordExpr{
			Kind:  "RecordExpr",
			Elems: out,
		}, nil
	case *ast.ArrayExpr:
		exprs, err := semExprs(scope, e.Exprs)
		if err != nil {
			return nil, err
		}
		return &dag.ArrayExpr{
			Kind:  "ArrayExpr",
			Exprs: exprs,
		}, nil
	case *ast.SetExpr:
		exprs, err := semExprs(scope, e.Exprs)
		if err != nil {
			return nil, err
		}
		return &dag.SetExpr{
			Kind:  "SetExpr",
			Exprs: exprs,
		}, nil
	case *ast.MapExpr:
		var entries []dag.Entry
		for _, entry := range e.Entries {
			key, err := semExpr(scope, entry.Key)
			if err != nil {
				return nil, err
			}
			val, err := semExpr(scope, entry.Value)
			if err != nil {
				return nil, err
			}
			entries = append(entries, dag.Entry{key, val})
		}
		return &dag.MapExpr{
			Kind:    "MapExpr",
			Entries: entries,
		}, nil
	}
	return nil, fmt.Errorf("invalid expression type %T", e)
}

func semID(scope *Scope, id *ast.ID) dag.Expr {
	// We use static scoping here to see if an identifier is
	// a "var" reference to the name or a field access
	// and transform the AST node appropriately.  The resulting DAG
	// doesn't have Identifiers as they are resolved here
	// one way or the other.
	if ref := scope.Lookup(id.Name); ref != nil {
		return ref
	}
	return pathOf(id.Name)
}

func semDynamicType(scope *Scope, tv astzed.Type) *dag.Call {
	if typeName, ok := tv.(*astzed.TypeName); ok {
		return dynamicTypeName(typeName.Name)
	}
	return nil
}

func dynamicTypeName(name string) *dag.Call {
	return &dag.Call{
		Kind: "Call",
		Name: "typename",
		Args: []dag.Expr{
			// ZSON string literal of type name
			&dag.Literal{
				Kind:  "Literal",
				Value: `"` + name + `"`,
			},
		},
	}
}

func semBinary(scope *Scope, e *ast.BinaryExpr) (dag.Expr, error) {
	op := e.Op
	if op == "." {
		lhs, err := semExpr(scope, e.LHS)
		if err != nil {
			return nil, err
		}
		id, ok := e.RHS.(*ast.ID)
		if !ok {
			return nil, errors.New("RHS of dot operator is not an identifier")
		}
		if lhs, ok := lhs.(*dag.This); ok {
			lhs.Path = append(lhs.Path, id.Name)
			return lhs, nil
		}
		return &dag.Dot{
			Kind: "Dot",
			LHS:  lhs,
			RHS:  id.Name,
		}, nil
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
		return &dag.BinaryExpr{
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
	// If we index this with a string constant, then just
	// extend the path.
	if op == "[" {
		if path := isIndexOfThis(scope, lhs, rhs); path != nil {
			return path, nil
		}
	}
	return &dag.BinaryExpr{
		Kind: "BinaryExpr",
		Op:   e.Op,
		LHS:  lhs,
		RHS:  rhs,
	}, nil
}

//XXX this should work for any path not just this, e.g., this.x["@foo"]
func isIndexOfThis(scope *Scope, lhs, rhs dag.Expr) *dag.This {
	if this, ok := lhs.(*dag.This); ok && len(this.Path) == 0 {
		if s, ok := isStringConst(scope, rhs); ok {
			this.Path = append(this.Path, s)
			return this
		}
	}
	return nil
}

func isStringConst(scope *Scope, e dag.Expr) (field string, ok bool) {
	val, err := kernel.EvalAtCompileTime(scope.zctx, e)
	if err == nil && val != nil && zed.TypeUnder(val.Type) == zed.TypeString {
		return string(val.Bytes), true
	}
	return "", false
}

func semSlice(scope *Scope, slice *ast.BinaryExpr) (*dag.BinaryExpr, error) {
	sliceFrom, err := semExprNullable(scope, slice.LHS)
	if err != nil {
		return nil, err
	}
	sliceTo, err := semExprNullable(scope, slice.RHS)
	if err != nil {
		return nil, err
	}
	return &dag.BinaryExpr{
		Kind: "BinaryExpr",
		Op:   ":",
		LHS:  sliceFrom,
		RHS:  sliceTo,
	}, nil
}

func semExprNullable(scope *Scope, e ast.Expr) (dag.Expr, error) {
	if e == nil {
		return nil, nil
	}
	return semExpr(scope, e)
}

func semCall(scope *Scope, call *ast.Call) (dag.Expr, error) {
	if e, err := maybeConvertAgg(scope, call); e != nil || err != nil {
		return e, err
	}
	if call.Where != nil {
		return nil, fmt.Errorf("'where' clause on non-aggregation function: %s", call.Name)
	}
	exprs, err := semExprs(scope, call.Args)
	if err != nil {
		return nil, fmt.Errorf("%s: bad argument: %w", call.Name, err)
	}
	return &dag.Call{
		Kind: "Call",
		Name: call.Name,
		Args: exprs,
	}, nil
}

func semExprs(scope *Scope, in []ast.Expr) ([]dag.Expr, error) {
	exprs := make([]dag.Expr, 0, len(in))
	for _, e := range in {
		expr, err := semExpr(scope, e)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}
	return exprs, nil
}

func semAssignments(scope *Scope, assignments []ast.Assignment, summarize bool) ([]dag.Assignment, error) {
	out := make([]dag.Assignment, 0, len(assignments))
	for _, e := range assignments {
		a, err := semAssignment(scope, e, summarize)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func semAssignment(scope *Scope, a ast.Assignment, summarize bool) (dag.Assignment, error) {
	rhs, err := semExpr(scope, a.RHS)
	if err != nil {
		return dag.Assignment{}, fmt.Errorf("rhs of assignment expression: %w", err)
	}
	if _, ok := rhs.(*dag.Agg); ok {
		summarize = true
	}
	var lhs dag.Expr
	if a.LHS != nil {
		lhs, err = semExpr(scope, a.LHS)
		if err != nil {
			return dag.Assignment{}, fmt.Errorf("lhs of assigment expression: %w", err)
		}
	} else if call, ok := a.RHS.(*ast.Call); ok {
		// If LHS is nil and the call is every() make the LHS field ts since
		// field ts assumed with every.
		name := call.Name
		if name == "every" {
			name = "ts"
		}
		lhs = &dag.This{"This", []string{name}}
	} else if agg, ok := a.RHS.(*ast.Agg); ok {
		lhs = &dag.This{"This", []string{agg.Name}}
	} else if v, ok := rhs.(*dag.Var); ok {
		lhs = &dag.This{"This", []string{v.Name}}
	} else {
		lhs, err = semExpr(scope, a.RHS)
		if err != nil {
			return dag.Assignment{}, errors.New("assignment name could not be inferred from rhs expression")
		}
	}
	if summarize {
		// Summarize always outputs its results as new records of "this"
		// so if we have an "as" that overrides "this", we just
		// convert it back to a local this.
		if dot, ok := lhs.(*dag.Dot); ok {
			if v, ok := dot.LHS.(*dag.Var); ok && v.Name == "this" {
				lhs = &dag.This{Kind: "This", Path: []string{dot.RHS}}
			}
		}
	}
	// Make sure we have a valid lval for lhs.
	this, ok := lhs.(*dag.This)
	if !ok {
		return dag.Assignment{}, errors.New("illegal left-hand side of assignment'")
	}
	if len(this.Path) == 0 {
		return dag.Assignment{}, errors.New("cannot assign to 'this'")
	}
	return dag.Assignment{"Assignment", lhs, rhs}, nil
}

func isThis(e ast.Expr) bool {
	if id, ok := e.(*ast.ID); ok {
		return id.Name == "this"
	}
	return false
}

func semFields(scope *Scope, exprs []ast.Expr) ([]dag.Expr, error) {
	fields := make([]dag.Expr, 0, len(exprs))
	for _, e := range exprs {
		f, err := semField(scope, e)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	return fields, nil
}

// semField analyzes the expression f and makes sure that it's
// a field reference returning an error if not.
func semField(scope *Scope, f ast.Expr) (*dag.This, error) {
	e, err := semExpr(scope, f)
	if err != nil {
		return nil, errors.New("invalid expression used as a field")
	}
	field, ok := e.(*dag.This)
	if !ok {
		return nil, errors.New("invalid expression used as a field")
	}
	if len(field.Path) == 0 {
		return nil, errors.New("cannot use 'this' as a field reference")
	}
	return field, nil
}

func convertCallProc(scope *Scope, call *ast.Call) (dag.Op, error) {
	agg, err := maybeConvertAgg(scope, call)
	if err != nil || agg != nil {
		return &dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{
				{
					Kind: "Assignment",
					LHS:  &dag.This{"This", field.New(call.Name)},
					RHS:  agg,
				},
			},
		}, err
	}
	if !function.HasBoolResult(call.Name) {
		return nil, fmt.Errorf("bad expression in filter: function %q does not return a boolean value", call.Name)
	}
	c, err := semCall(scope, call)
	if err != nil {
		return nil, err
	}
	return &dag.Filter{
		Kind: "Filter",
		Expr: c,
	}, nil
}

func maybeConvertAgg(scope *Scope, call *ast.Call) (dag.Expr, error) {
	if _, err := agg.NewPattern(call.Name); err != nil {
		return nil, nil
	}
	var e dag.Expr
	if len(call.Args) > 1 {
		if call.Name == "min" || call.Name == "max" {
			// min and max are special cases as they are also functions. If the
			// number of args is greater than 1 they're probably a function so do not
			// return an error.
			return nil, nil
		}
		return nil, fmt.Errorf("%s: wrong number of arguments", call.Name)
	}
	if len(call.Args) == 1 {
		var err error
		e, err = semExpr(scope, call.Args[0])
		if err != nil {
			return nil, err
		}
	}
	where, err := semExprNullable(scope, call.Where)
	if err != nil {
		return nil, err
	}
	return &dag.Agg{
		Kind:  "Agg",
		Name:  call.Name,
		Expr:  e,
		Where: where,
	}, nil
}

func DotExprToFieldPath(e ast.Expr) *dag.This {
	switch e := e.(type) {
	case *ast.BinaryExpr:
		if e.Op == "." {
			lhs := DotExprToFieldPath(e.LHS)
			if lhs == nil {
				return nil
			}
			id, ok := e.RHS.(*ast.ID)
			if !ok {
				return nil
			}
			lhs.Path = append(lhs.Path, id.Name)
			return lhs
		}
		if e.Op == "[" {
			lhs := DotExprToFieldPath(e.LHS)
			if lhs == nil {
				return nil
			}
			id, ok := e.RHS.(*astzed.Primitive)
			if !ok || id.Type != "string" {
				return nil
			}
			lhs.Path = append(lhs.Path, id.Text)
			return lhs
		}
	case *ast.ID:
		return pathOf(e.Name)
	}
	// This includes a null Expr, which can happen if the AST is missing
	// a field or sets it to null.
	return nil
}

func pathOf(name string) *dag.This {
	var path []string
	if name != "this" {
		path = []string{name}
	}
	return &dag.This{Kind: "This", Path: path}
}

func semType(scope *Scope, typ astzed.Type) (string, error) {
	ztype, err := zson.TranslateType(scope.zctx, typ)
	if err != nil {
		return "", err
	}
	return zson.FormatType(ztype), nil
}
