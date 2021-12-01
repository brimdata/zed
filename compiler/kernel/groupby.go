package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/groupby"
)

func compileGroupBy(pctx *proc.Context, scope *Scope, parent proc.Interface, summarize *dag.Summarize) (*groupby.Proc, error) {
	keys, err := compileAssignments(summarize.Keys, pctx.Zctx, scope)
	if err != nil {
		return nil, err
	}
	names, reducers, err := compileAggAssignments(summarize.Aggs, scope, pctx.Zctx)
	if err != nil {
		return nil, err
	}
	dir := order.Direction(summarize.InputSortDir)
	return groupby.New(pctx, parent, keys, names, reducers, summarize.Limit, dir, summarize.PartialsIn, summarize.PartialsOut)
}

func compileAggAssignments(assignments []dag.Assignment, scope *Scope, zctx *zed.Context) (field.List, []*expr.Aggregator, error) {
	names := make(field.List, 0, len(assignments))
	aggs := make([]*expr.Aggregator, 0, len(assignments))
	for _, assignment := range assignments {
		name, agg, err := compileAggAssignment(zctx, scope, assignment)
		if err != nil {
			return nil, nil, err
		}
		aggs = append(aggs, agg)
		names = append(names, name)
	}
	return names, aggs, nil
}

func compileAggAssignment(zctx *zed.Context, scope *Scope, assignment dag.Assignment) (field.Path, *expr.Aggregator, error) {
	aggAST, ok := assignment.RHS.(*dag.Agg)
	if !ok {
		return nil, nil, errors.New("aggregator is not an aggregation expression")
	}
	lhs, err := compileLval(assignment.LHS)
	if err != nil {
		return nil, nil, fmt.Errorf("lhs of aggregation: %w", err)
	}
	m, err := compileAgg(zctx, scope, aggAST)
	return lhs, m, err
}

func compileAgg(zctx *zed.Context, scope *Scope, agg *dag.Agg) (*expr.Aggregator, error) {
	name := agg.Name
	var err error
	var arg expr.Evaluator
	if agg.Expr != nil {
		arg, err = compileExpr(zctx, nil, agg.Expr)
		if err != nil {
			return nil, err
		}
	}
	var where expr.Filter
	if agg.Where != nil {
		where, err = CompileFilter(zctx, scope, agg.Where)
		if err != nil {
			return nil, err
		}
	}
	return expr.NewAggregator(name, arg, where)
}
