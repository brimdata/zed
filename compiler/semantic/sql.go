package semantic

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/expr/agg"
	"github.com/brimdata/zed/field"
)

func convertSQLProc(scope *Scope, sql *ast.SQLExpr) (ast.Proc, error) {
	selection, err := newSQLSelection(scope, sql.Select)
	if err != err {
		return nil, err
	}
	var procs []ast.Proc
	if sql.From != nil {
		tableFilter, err := convertSQLTableRef(scope, sql.From.Table)
		if err != nil {
			return nil, err
		}
		procs = append(procs, tableFilter)
		if sql.From.Alias != nil {
			// If there's an alias, we do a 'cut alias=.'
			alias, err := convertSQLAlias(scope, sql.From.Alias)
			if err != nil {
				return nil, err
			}
			// If the FROM table has been aliased and all join clauses, if any,
			// are also aliased, then we can lift any where claused that is dependent
			// only on the FROM table or is a component of a logical AND and that
			// component depends only on the FROM table.  In this case, we splice
			// the filter predicate after the FROM expression and before everything else.
			// This is a "peephole" optimization that will go away once we
			// have fully fledge data-flow-based optimizations.
			if sql.Where != nil {
				if f := liftWhereFilter(sql.From.Alias, sql.Where, sql.Joins); f != nil {
					procs = append(procs, f)
				}
			}
			procs = append(procs, alias)
		}
	}
	if sql.Joins != nil {
		if len(procs) == 0 {
			return nil, errors.New("cannot JOIN without a FROM")
		}
		procs, err = convertSQLJoins(scope, procs, sql.Joins)
		if err != nil {
			return nil, err
		}
	}
	if sql.Where != nil {
		filter := &ast.Filter{
			Kind: "Filter",
			Expr: sql.Where,
		}
		procs = append(procs, filter)
	}
	if sql.GroupBy != nil {
		groupby, err := convertSQLGroupBy(scope, sql.GroupBy, selection)
		if err != nil {
			return nil, err
		}
		procs = append(procs, groupby)
		if sql.Having != nil {
			having, err := semExpr(scope, sql.Having)
			if err != nil {
				return nil, err
			}
			filter := &ast.Filter{
				Kind: "Filter",
				Expr: having,
			}
			procs = append(procs, filter)
		}
	} else if sql.Select != nil {
		if sql.Having != nil {
			return nil, errors.New("HAVING clause used without GROUP BY")
		}
		// GroupBy will do the cutting but if there's no GroupBy,
		// then we need a cut for the select expressions.
		// For SELECT *, cutter is nil.
		selector, err := convertSQLSelect(selection)
		if err != nil {
			return nil, err
		}
		if selector != nil {
			procs = append(procs, selector)
		}
	}
	if sql.OrderBy != nil {
		procs = append(procs, sortByMulti(sql.OrderBy.Keys, sql.OrderBy.Order))
	}
	if sql.Limit != 0 {
		p := &ast.Head{
			Kind:  "Head",
			Count: sql.Limit,
		}
		procs = append(procs, p)
	}
	if len(procs) == 0 {
		procs = []ast.Proc{passProc}
	}
	return wrap(procs), nil
}

func isId(e ast.Expr) (string, bool) {
	if id, ok := e.(*ast.Id); ok {
		return id.Name, true
	}
	return "", false
}

func liftWhereFilter(alias, where ast.Expr, joins []ast.SQLJoin) *ast.Filter {
	aliasId, ok := isId(alias)
	if !ok {
		return nil
	}
	for _, join := range joins {
		if _, ok := isId(join.Alias); !ok {
			return nil
		}
	}
	eligible := eligiblePred(aliasId, where)
	if eligible == nil {
		return nil
	}
	return &ast.Filter{
		Kind: "Filter",
		Expr: eligible,
	}
}

func eligiblePred(aliasId string, e ast.Expr) ast.Expr {
	switch e := e.(type) {
	case *ast.UnaryExpr:
		if operand := eligiblePred(aliasId, e.Operand); operand != nil {
			return &ast.UnaryExpr{
				Kind:    "UnaryExpr",
				Op:      "!",
				Operand: operand,
			}
		}
	case *ast.Primitive:
		return e
	case *ast.BinaryExpr:
		if e.Op == "." {
			return eligibleId(aliasId, e)
		}
		lhs := eligiblePred(aliasId, e.LHS)
		rhs := eligiblePred(aliasId, e.RHS)
		if e.Op == "or" {
			if lhs != nil && rhs != nil {
				return e
			}
			return nil
		}
		if e.Op == "and" {
			if lhs == nil {
				return rhs
			}
			if rhs == nil {
				return lhs
			}
			return e
		}
		if lhs != nil && rhs != nil {
			return &ast.BinaryExpr{
				Kind: "BinaryExpr",
				Op:   e.Op,
				LHS:  lhs,
				RHS:  rhs,
			}
		}
	}
	return nil
}

func eligibleId(aliasId string, e *ast.BinaryExpr) ast.Expr {
	if id, ok := e.LHS.(*ast.Id); ok && id.Name == aliasId {
		return e.RHS
	}
	return nil
}

func convertSQLTableRef(scope *Scope, e ast.Expr) (ast.Proc, error) {
	// If an identifier name is given with no definition for that name,
	// then convert it to a type name as it is otherwise expected that
	// the type name will be defined by the data stream.
	if id, ok := e.(*ast.Id); ok && scope.Lookup(id.Name) == nil {
		e = &ast.TypeValue{
			Kind: "TypeValue",
			Value: &ast.TypeName{
				Kind: "TypeName",
				Name: id.Name,
			},
		}
	}
	return &ast.Call{
		Kind: "Call",
		Name: "is",
		Args: []ast.Expr{e},
	}, nil
}

func convertSQLAlias(scope *Scope, e ast.Expr) (*ast.Cut, error) {
	if _, err := semField(scope, e); err != nil {
		return nil, fmt.Errorf("illegal alias: %w", err)
	}
	cut := ast.Assignment{
		Kind: "Assignment",
		LHS:  e,
		RHS:  &ast.Root{},
	}
	return &ast.Cut{
		Kind: "Cut",
		Args: []ast.Assignment{cut},
	}, nil
}

func wrap(procs []ast.Proc) ast.Proc {
	if len(procs) == 0 {
		return nil
	}
	if len(procs) == 1 {
		return procs[0]
	}
	return &ast.Sequential{
		Kind:  "Sequential",
		Procs: procs,
	}
}

func convertSQLJoins(scope *Scope, fromPath []ast.Proc, joins []ast.SQLJoin) ([]ast.Proc, error) {
	left := fromPath
	for _, right := range joins {
		var err error
		left, err = convertSQLJoin(scope, left, right)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

// For now, each joining table is on the right...
// We don't have logic to not care about the side of the JOIN ON keys...
func convertSQLJoin(scope *Scope, leftPath []ast.Proc, sqlJoin ast.SQLJoin) ([]ast.Proc, error) {
	if sqlJoin.Alias == nil {
		return nil, errors.New("JOIN currently requires alias, e.g., JOIN <type> <alias> (will be fixed soon)")
	}
	leftPath = append(leftPath, sortBy(sqlJoin.LeftKey))

	joinFilter, err := convertSQLTableRef(scope, sqlJoin.Table)
	if err != nil {
		return nil, err
	}
	rightPath := []ast.Proc{joinFilter}
	cut, err := convertSQLAlias(scope, sqlJoin.Alias)
	if err != nil {
		return nil, errors.New("JOIN alias must be a name")
	}
	rightPath = append(rightPath, cut)
	rightPath = append(rightPath, sortBy(sqlJoin.RightKey))

	fork := &ast.Parallel{
		Kind:  "Parallel",
		Procs: []ast.Proc{wrap(leftPath), wrap(rightPath)},
	}
	alias := ast.Assignment{
		Kind: "Assignment",
		RHS:  sqlJoin.Alias,
	}
	join := &ast.Join{
		Kind:     "Join",
		Style:    sqlJoin.Style,
		LeftKey:  sqlJoin.LeftKey,
		RightKey: sqlJoin.RightKey,
		Args:     []ast.Assignment{alias},
	}
	return []ast.Proc{fork, join}, nil
}

func sortBy(e ast.Expr) *ast.Sort {
	return sortByMulti([]ast.Expr{e}, "asc")
}

func sortByMulti(keys []ast.Expr, order string) *ast.Sort {
	// XXX ast.Sort should take a zbuf.Order instead of an in direction
	// (and probably this constant should move out of zbuf and into ast)
	// See issue #2397
	direction := 1
	if order == "desc" {
		direction = -1
	}
	return &ast.Sort{
		Kind:    "Sort",
		Args:    keys,
		SortDir: direction,
	}
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
		return selection.cut(), nil
	}
	if nagg != len(selection) {
		return nil, errors.New("cannot mix aggregations and non-aggregations without a GROUP BY")
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
			Kind: "Assignment",
			LHS:  p.assignment.LHS,
			RHS:  p.agg,
		}
		assignments = append(assignments, a)
	}
	return &ast.Summarize{
		Kind: "Summarize",
		Aggs: assignments,
	}, nil
}

func convertSQLGroupBy(scope *Scope, groupByKeys []ast.Expr, selection sqlSelection) (ast.Proc, error) {
	var keys []field.Static
	for _, key := range groupByKeys {
		name, err := sqlField(scope, key)
		if err != nil {
			return nil, fmt.Errorf("bad GROUP BY key: %w", err)
		}
		keys = append(keys, name)
	}
	// Make sure all group-by keys are in the selection.
	all := selection.fields()
	for _, key := range keys {
		//XXX fix this for select *?
		if !key.In(all) {
			if key.HasPrefixIn(all) {
				return nil, fmt.Errorf("'%s': GROUP BY key cannot be a sub-field of the selected value", key)
			}
			return nil, fmt.Errorf("'%s': GROUP BY key not in selection", key)
		}
	}
	// Make sure all scalars are in the group-by keys.
	scalars := selection.scalars()
	for _, f := range scalars.fields() {
		if !f.In(keys) {
			return nil, fmt.Errorf("'%s': selected expression is missing from GROUP BY clause (and is not an aggregation)", f)
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
	for _, p := range selection.aggs() {
		aggExprs = append(aggExprs, ast.Assignment{
			LHS: p.assignment.LHS,
			RHS: p.agg,
		})
	}
	// XXX how to override limit for spills?
	return &ast.Summarize{
		Kind: "Summarize",
		Keys: keyExprs,
		Aggs: aggExprs,
	}, nil
}

// A sqlPick is one column of a select statement.  We bookkeep here whether
// a column is a scalar expression or an aggregation by looking up the function
// name and seeing if it's an aggregator or not.  We also infer the column
// names so we can do SQL error checking relating the selections to the group-by
// keys, something that is not needed in Z.
type sqlPick struct {
	name       field.Static
	agg        *ast.Agg
	assignment ast.Assignment
}

type sqlSelection []sqlPick

func newSQLSelection(scope *Scope, assignments []ast.Assignment) (sqlSelection, error) {
	// Make a cut from a SQL select.  This should just work
	// without having to track identifier names of columns because
	// the transformations will all relable data from stage to stage
	// and Select names refer to the names at the last stage of
	// the table.
	var s sqlSelection
	for _, a := range assignments {
		name, err := deriveAs(scope, a)
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

func (s sqlSelection) fields() []field.Static {
	var fields []field.Static
	for _, p := range s {
		fields = append(fields, p.name)
	}
	return fields
}

func (s sqlSelection) aggs() sqlSelection {
	var aggs sqlSelection
	for _, p := range s {
		if p.agg != nil {
			aggs = append(aggs, p)
		}
	}
	return aggs
}

func (s sqlSelection) scalars() sqlSelection {
	var scalars sqlSelection
	for _, p := range s {
		if p.agg == nil {
			scalars = append(scalars, p)
		}
	}
	return scalars
}

func (s sqlSelection) cut() *ast.Cut {
	if len(s) == 0 {
		return nil
	}
	var a []ast.Assignment
	for _, p := range s {
		a = append(a, p.assignment)
	}
	return &ast.Cut{
		Kind: "Cut",
		Args: a,
	}
}

func isAgg(e ast.Expr) (*ast.Agg, error) {
	call, ok := e.(*ast.Call)
	if !ok {
		return nil, nil
	}
	if _, err := agg.NewPattern(call.Name); err != nil {
		return nil, nil
	}
	var arg ast.Expr
	if len(call.Args) > 1 {
		return nil, fmt.Errorf("%s: wrong number of arguments", call.Name)
	}
	if len(call.Args) == 1 {
		arg = call.Args[0]
	}
	return &ast.Agg{
		Kind: "Agg",
		Name: call.Name,
		Expr: arg,
	}, nil
}

func deriveAs(scope *Scope, a ast.Assignment) (field.Static, error) {
	sa, err := semAssignment(scope, a)
	if err != nil {
		return nil, fmt.Errorf("AS clause of SELECT: %w", err)
	}
	f, ok := sa.LHS.(*ast.Path)
	if !ok {
		return nil, fmt.Errorf("AS clause not a field: %w", err)
	}
	return f.Name, nil
}

func sqlField(scope *Scope, e ast.Expr) (field.Static, error) {
	name, err := semField(scope, e)
	if err != nil {
		return nil, err
	}
	if f, ok := name.(*ast.Path); ok {
		return f.Name, nil
	}
	return nil, errors.New("expression is not a field reference")
}
