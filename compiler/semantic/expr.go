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
		// We use static scoping here to see if an identifier is
		// a "var" reference to the name or a field access
		// and transform the ast node appropriately.  The semantic AST
		// should have no Identifiers in it as they should all be
		// resolved one way or another.
		if ref := scope.Lookup(e.Name); ref != nil {
			return ref, nil
		}
		return semField(scope, e)
	case *ast.This:
		return semField(scope, e)
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
			// the literal reference with a type_by_name() call.
			// Note that we just check the top value here but there can be
			// nested dynamic type references inside a complex type but this
			// is not yet supported and will fail here with a compile-time error
			// complaining about the type not existing.
			// XXX See issue #3413
			if e := semDynamicType(scope, e.Value); e != nil {
				return e, nil
			}
			return nil, err
		}
		return &dag.Literal{
			Kind: "Literal",
			//XXX flag week: this changes to angle brackets
			Value: "(" + typ + ")",
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
		var fields []dag.Field
		for _, f := range e.Fields {
			value, err := semExpr(scope, f.Value)
			if err != nil {
				return nil, err
			}
			fields = append(fields, dag.Field{f.Name, value})
		}
		return &dag.RecordExpr{
			Kind:   "RecordExpr",
			Fields: fields,
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
func isIndexOfThis(scope *Scope, lhs, rhs dag.Expr) *dag.Path {
	if path, ok := lhs.(*dag.Path); ok && len(path.Name) == 0 {
		if s, ok := isStringConst(scope, rhs); ok {
			path.Name = append(path.Name, s)
			return path
		}
	}
	return nil
}

func isStringConst(scope *Scope, e dag.Expr) (field string, ok bool) {
	val, err := kernel.EvalAtCompileTime(scope.zctx, e)
	if err == nil && val != nil && val.Type == zed.TypeString {
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

func semAssignments(scope *Scope, assignments []ast.Assignment) ([]dag.Assignment, error) {
	out := make([]dag.Assignment, 0, len(assignments))
	for _, e := range assignments {
		a, err := semAssignment(scope, e)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func semAssignment(scope *Scope, a ast.Assignment) (dag.Assignment, error) {
	rhs, err := semExpr(scope, a.RHS)
	if err != nil {
		return dag.Assignment{}, fmt.Errorf("rhs of assignment expression: %w", err)
	}
	var lhs dag.Expr
	if a.LHS != nil {
		// XXX currently only support explicit field lvals
		// (i.e., no assignments to array elements etc... instead
		// you create a new array with modified contends)
		lhs, err = semField(scope, a.LHS)
		if err != nil {
			return dag.Assignment{}, fmt.Errorf("lhs of assigment expression: %w", err)
		}
	} else if call, ok := a.RHS.(*ast.Call); ok {
		lhs = &dag.Path{"Path", []string{call.Name}}
	} else if agg, ok := a.RHS.(*ast.Agg); ok {
		lhs = &dag.Path{"Path", []string{agg.Name}}
	} else if _, ok := a.RHS.(*ast.This); ok {
		return dag.Assignment{}, errors.New(`cannot assign to "this"`)
	} else {
		lhs, err = semField(scope, a.RHS)
		if err != nil {
			return dag.Assignment{}, errors.New("assignment name could not be inferred from rhs expression")
		}
	}
	return dag.Assignment{"Assignment", lhs, rhs}, nil
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

// semField checks that an expression is a field refernce and converts it
// to a field path if possible.  It will convert any references to Refs.
func semField(scope *Scope, e ast.Expr) (dag.Expr, error) {
	switch e := e.(type) {
	case *ast.BinaryExpr:
		if e.Op == "." {
			lhs, err := semField(scope, e.LHS)
			if err != nil {
				return nil, err
			}
			id, ok := e.RHS.(*ast.ID)
			if !ok {
				return nil, errors.New("RHS of dot operator is not an identifier")
			}
			if lhs, ok := lhs.(*dag.Path); ok {
				lhs.Name = append(lhs.Name, id.Name)
				return lhs, nil
			}
			return &dag.Dot{
				Kind: "Dot",
				LHS:  lhs,
				RHS:  id.Name,
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
			if path := isIndexOfThis(scope, lhs, rhs); path != nil {
				return path, nil
			}
			return &dag.BinaryExpr{
				Kind: "BinaryExpr",
				Op:   "[",
				LHS:  lhs,
				RHS:  rhs,
			}, nil
		}
	case *ast.ID:
		if ref := scope.Lookup(e.Name); ref != nil {
			return ref, nil
		}
		return &dag.Path{Kind: "Path", Name: []string{e.Name}}, nil
	case *ast.This:
		//XXX nil? empty path?
		return &dag.Path{Kind: "Path", Name: []string{}}, nil
	}
	// This includes a null Expr, which can happen if the AST is missing
	// a field or sets it to null.
	return nil, errors.New("expression is not a field reference.")
}

func convertCallProc(scope *Scope, call *ast.Call) (dag.Op, error) {
	agg, err := maybeConvertAgg(scope, call)
	if err != nil || agg != nil {
		return &dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{
				{
					Kind: "Assignment",
					LHS:  &dag.Path{"Path", field.New(call.Name)},
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

func DotExprToFieldPath(e ast.Expr) *dag.Path {
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
			lhs.Name = append(lhs.Name, id.Name)
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
			lhs.Name = append(lhs.Name, id.Text)
			return lhs
		}
	case *ast.ID:
		return &dag.Path{Kind: "Path", Name: []string{e.Name}}
	case *ast.This:
		return &dag.Path{Kind: "Path", Name: []string{}}
	}
	// This includes a null Expr, which can happen if the AST is missing
	// a field or sets it to null.
	return nil
}

func semType(scope *Scope, typ astzed.Type) (string, error) {
	ztype, err := zson.TranslateType(scope.zctx, typ)
	if err != nil {
		return "", err
	}
	return zson.FormatType(ztype), nil
}
