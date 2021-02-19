package kernel

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/groupby"
	"github.com/brimsec/zq/zng/resolver"
)

func compileAgg(pctx *proc.Context, scope *Scope, parent proc.Interface, p *Agg) (*groupby.Proc, error) {
	keys, err := compileAssignments(p.Keys, pctx.TypeContext, scope) //XXX
	if err != nil {
		return nil, err
	}
	aggLvals, aggFuncs, err := compileAggAssignments(p.AggAssignments, scope, pctx.TypeContext)
	if err != nil {
		return nil, err
	}
	sortdir := 1
	if p.InputDesc {
		sortdir = -1
	}
	return groupby.New(pctx, parent, keys, lvals, aggFuncs, p.Limit, sortdir, p.partialsIn, p.partialsIn)
}

func compileAggAssignments(assignments []AggAssignment, zctx *resolver.Context, scope *Scope) ([]field.Static, []expr.Evaluator, error) {
	lvals := make([]field.Static, 0, len(assignments))
	exprs := make([]expr.Evaluator, 0, len(assignments))
	for _, assignment := range assignments {
		e, err := compileExpr(zctx, scope, &assignment.RHS)
		if err != nil {
			return nil, nil, err
		}
		exprs = append(exprs, e)
		lval, err := CompileLval(assignment.LHS)
		if err != nil {
			// XXX this error checking shouldn't be necessary
			return nil, nil, fmt.Errorf("lhs of assigment agg assignment: %w", err)
		}
	}
	return lvals, exprs, nil
}

func compileAggFuncs(assignments []ast.Assignment, scope *Scope, zctx *resolver.Context) ([]field.Static, []*expr.Aggregator, error) {
	names := make([]field.Static, 0, len(assignments))
	aggs := make([]*expr.Aggregator, 0, len(assignments))
	for _, assignment := range assignments {
		name, agg, err := compileAgg(zctx, scope, assignment)
		if err != nil {
			return nil, nil, err
		}
		aggs = append(aggs, agg)
		names = append(names, name)
	}
	return names, aggs, nil
}

func compileAgg(zctx *resolver.Context, scope *Scope, assignment ast.Assignment) (field.Static, *expr.Aggregator, error) {
	aggAST, ok := assignment.RHS.(*ast.Reducer)
	if !ok {
		return nil, nil, errors.New("aggregator is not an aggregation expression")
	}
	aggOp := aggAST.Operator
	var err error
	var arg expr.Evaluator
	if aggAST.Expr != nil {
		arg, err = compileExpr(zctx, nil, aggAST.Expr)
		if err != nil {
			return nil, nil, err
		}
	}
	// If there is a reducer assignment, the LHS is non-nil and we
	// compile.  Otherwise, we infer an LHS top-level field name from
	// the name of reducer function.
	var lhs field.Static
	if assignment.LHS == nil {
		lhs = field.New(aggOp)
	} else {
		lhs, err = CompileLval(assignment.LHS)
		if err != nil {
			return nil, nil, fmt.Errorf("lhs of aggregation: %w", err)
		}
	}
	var where expr.Filter
	if aggAST.Where != nil {
		where, err = compileFilter(zctx, scope, aggAST.Where)
		if err != nil {
			return nil, nil, err
		}
	}
	m, err := expr.NewAggregator(aggOp, arg, where)
	return lhs, m, err
}
