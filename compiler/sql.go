package compiler

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr/agg"
	"github.com/brimsec/zq/field"
)

func convertSQLProc(sql *ast.SqlExpression) (ast.Proc, error) {
	selection, err := newSQLSelection(sql.Select)
	if err != err {
		return nil, err
	}
	var procs []ast.Proc
	if sql.From != nil {
		tableFilter, err := convertSQLTableRef(sql.From.Table)
		if err != nil {
			return nil, err
		}
		procs = append(procs, tableFilter)
		if sql.From.Alias != nil {
			// If there's an alias, we do a 'cut alias=.'
			alias, err := convertSQLAlias(sql.From.Alias)
			if err != nil {
				return nil, err
			}
			procs = append(procs, alias)
		}
	}
	if sql.Where != nil {
		filter := &ast.FilterProc{
			Op:     "FilterProc",
			Filter: sql.Where,
		}
		procs = append(procs, filter)
	}
	if sql.GroupBy != nil {
		groupby, err := convertSQLGroupBy(sql.GroupBy, selection)
		if err != nil {
			return nil, err
		}
		procs = append(procs, groupby)
		if sql.Having != nil {
			filter := &ast.FilterProc{
				Op:     "FilterProc",
				Filter: sql.Having,
			}
			procs = append(procs, filter)
		}
	} else if sql.Select != nil {
		if sql.Having != nil {
			return nil, errors.New("HAVING clause used without GROUP BY")
		}
		selector, err := convertSQLSelect(selection)
		if err != nil {
			return nil, err
		}
		// GroupBy will do the cutting but if there's no GroupBy,
		// then we need a cut for the select expressions.
		// For SELECT *, cutter is nil.
		procs = append(procs, selector)
	}
	if sql.Limit != 0 {
		p := &ast.HeadProc{
			Op:    "HeadProc",
			Count: sql.Limit,
		}
		procs = append(procs, p)
	}
	if len(procs) == 0 {
		return nil, nil
	}
	if len(procs) == 1 {
		return procs[0], nil
	}
	return &ast.SequentialProc{
		Op:    "SequentialProc",
		Procs: procs,
	}, nil
}

func convertSQLTableRef(e ast.Expression) (ast.Proc, error) {
	// For now, we special case a string that parses as a ZSON type.
	// If not, we try to compiler this as a filter expression.
	if literal, ok := e.(*ast.Literal); ok && literal.Type == "string" {
		e = &ast.BinaryExpression{
			Op:       "BinaryExpr",
			Operator: "=",
			LHS: &ast.FunctionCall{
				Op:       "FunctionCall",
				Function: "typeof",
				Args: []ast.Expression{
					&ast.RootRecord{},
				},
			},
			RHS: literal,
		}
	}
	return &ast.FilterProc{
		Op:     "FilterProc",
		Filter: e,
	}, nil
}

func convertSQLAlias(e ast.Expression) (*ast.CutProc, error) {
	if _, err := CompileLval(e); err != nil {
		return nil, fmt.Errorf("illegal alias: %w", err)
	}
	cut := ast.Assignment{
		Op:  "Assignment",
		LHS: e,
		RHS: &ast.RootRecord{},
	}
	return &ast.CutProc{
		Op:     "CutProc",
		Fields: []ast.Assignment{cut},
	}, nil
}

func convertSQLSelect(selection sqlSelection) (ast.Proc, error) {
	// This is a straight select without a group-by.
	// If all the expressions are aggregators, then we build a group-by.
	// If it's mixed, we return an error.  Otherwise, we do a simple cut.
	var nagg int
	for _, p := range selection {
		if p.agg != nil {
			nagg++
		}
	}
	if nagg == 0 {
		return selection.Cut(), nil
	}
	if nagg != len(selection) {
		return nil, errors.New("cannot mix aggregations and non-aggregations without a group-by")
	}
	// Note here that we reconstruct the group-by aggregators instead of
	// using the assignments in ast.SqlExpression.Select since the SQL peg
	// parser does not know whether they are aggregators or function calls,
	// but the sqlPick elements have this determined.  So we take the LHS
	// from the original expression and mix it with the agg that was put
	// in sqlPick.
	var assignments []ast.Assignment
	for _, p := range selection {
		a := ast.Assignment{
			Op:  "Assignment",
			LHS: p.assignment.LHS,
			RHS: p.agg,
		}
		assignments = append(assignments, a)
	}
	return &ast.GroupByProc{
		Op:       "GroupByProc",
		Reducers: assignments,
	}, nil
}

//XXX CompileLval -> deriveLvalField
// We can simplify this so deriveAs and deriveLvalField are mutually recursive
// in the proeper way, then we can back integrate this soltuion into the
// rest of the expression compiler

func convertSQLGroupBy(groupByKeys []ast.Expression, selection sqlSelection) (ast.Proc, error) {
	var keys []field.Static
	for _, key := range groupByKeys {
		name, err := CompileLval(key)
		if err != nil {
			return nil, fmt.Errorf("bad group-by key: %w", err)
		}
		keys = append(keys, name)
	}
	// Make sure all group-by keys are in the selection.
	all := selection.Fields()
	for _, key := range keys {
		//XXX fix this for select *?
		if !key.In(all) {
			if key.HasPrefixIn(all) {
				return nil, fmt.Errorf("'%s': group-by key cannot be a sub-field of the selected value", key)
			}
			return nil, fmt.Errorf("'%s': group-by key not in selection", key)
		}
	}
	// Make sure all scalars are in the group-by keys.
	scalars := selection.Scalars()
	for _, f := range scalars.Fields() {
		if !f.In(keys) {
			return nil, fmt.Errorf("'%s': selected expression is missing from group-by clause (and is not an aggregation)", f)
		}
	}
	// Now that the selection and keys have been checked, build the
	// key expressions from the scalars of the select and build the
	// aggregators (aka reducers) from the aggregation functions present
	// in the select clause.
	var keyExprs []ast.Assignment
	for _, p := range scalars {
		keyExprs = append(keyExprs, p.assignment)
	}
	var aggExprs []ast.Assignment
	for _, p := range selection.Aggs() {
		aggExprs = append(aggExprs, p.assignment)
	}
	// XXX how to override limit for spills?
	return &ast.GroupByProc{
		Op:       "GroupByProc",
		Keys:     keyExprs,
		Reducers: aggExprs,
	}, nil
}

// A sqlPick is one column of a XXX
type sqlPick struct {
	name       field.Static
	agg        *ast.Reducer // XXX not sure we need this because we can just copy from select
	assignment ast.Assignment
}

type sqlSelection []sqlPick

func newSQLSelection(assignments []ast.Assignment) (sqlSelection, error) {
	// Make a cut from a SQL select.  This should just work
	// without having to track identifier names of columns because
	// the transformations will all relable data from stage to stage
	// and Select names refer to the names at the last stage of
	// the table.

	// XXX we should do a semantic check of everything before we
	// traansform the AST, otherwise the user is going to get weird
	// errors (i.e., bad cut expression etc).  We will live with this
	// for now as we get this working.
	var s sqlSelection
	for _, a := range assignments {
		name, err := deriveAs(a)
		if err != nil {
			return nil, err
		}
		agg, err := isAgg(a.RHS)
		if err != nil {
			return nil, err
		}
		s = append(s, sqlPick{name, agg, a})
	}
	return s, nil
}

func (s sqlSelection) Fields() []field.Static {
	var fields []field.Static
	for _, p := range s {
		fields = append(fields, p.name)
	}
	return fields
}

func (s sqlSelection) Aggs() sqlSelection {
	var aggs sqlSelection
	for _, p := range s {
		if p.agg != nil {
			aggs = append(aggs, p)
		}
	}
	return aggs
}

func (s sqlSelection) Scalars() sqlSelection {
	var scalars sqlSelection
	for _, p := range s {
		if p.agg == nil {
			scalars = append(scalars, p)
		}
	}
	return scalars
}

func (s sqlSelection) Cut() *ast.CutProc {
	if len(s) == 0 {
		return nil
	}
	var a []ast.Assignment
	for _, p := range s {
		a = append(a, p.assignment)
	}
	return &ast.CutProc{
		Op:     "CutProc",
		Fields: a,
	}
}

func isAgg(e ast.Expression) (*ast.Reducer, error) {
	call, ok := e.(*ast.FunctionCall)
	if !ok {
		return nil, nil
	}
	if _, err := agg.NewPattern(call.Function); err != nil {
		return nil, nil
	}
	var arg ast.Expression
	if len(call.Args) > 1 {
		return nil, fmt.Errorf("%s: wrong number of arguments", call.Function)
	}
	if len(call.Args) == 1 {
		arg = call.Args[0]
	}
	return &ast.Reducer{
		Op:       "Reducer",
		Operator: call.Function,
		Expr:     arg,
	}, nil
}

func deriveAs(a ast.Assignment) (field.Static, error) {
	//XXX this logic should be shared with CompileAssignment
	if a.LHS != nil {
		f, err := CompileLval(a.LHS)
		if err != nil {
			return nil, fmt.Errorf("AS clause of select: %w", err)
		}
		return f, nil
	}
	switch rhs := a.RHS.(type) {
	case *ast.RootRecord:
		return field.New("."), nil
	case *ast.Identifier:
		return field.New(rhs.Name), nil
	case *ast.FunctionCall:
		return field.New(rhs.Function), nil
	case *ast.BinaryExpression:
		// This can be a dotted record or some other expression.
		// In the latter case, it might be nice to infer a name,
		// e.g., forr "count() by a+b" we could infer "sum" for
		// the name, i,e., "count() by sum=a+b".  But for now,
		// we'll just catch this as an error.
		f, err := CompileLval(rhs)
		if err == nil {
			return f, nil
		}
	}
	return nil, errors.New("cannot infer AS name of select")
}
