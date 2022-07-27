package querygen

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op/groupby"
	"github.com/brimdata/zed/zbuf"
)

func (b *Builder) compileGroupBy(parent zbuf.Puller, summarize *dag.Summarize) (*groupby.Proc, error) {
	keys, err := b.compileAssignments(summarize.Keys)
	if err != nil {
		return nil, err
	}
	names, reducers, err := b.compileAggAssignments(summarize.Aggs)
	if err != nil {
		return nil, err
	}
	dir := order.Direction(summarize.InputSortDir)
	return groupby.New(b.pctx, parent, keys, names, reducers, summarize.Limit, dir, summarize.PartialsIn, summarize.PartialsOut)
}

func (b *Builder) compileAggAssignments(assignments []dag.Assignment) (field.List, []*expr.Aggregator, error) {
	names := make(field.List, 0, len(assignments))
	aggs := make([]*expr.Aggregator, 0, len(assignments))
	for _, assignment := range assignments {
		name, agg, err := b.compileAggAssignment(assignment)
		if err != nil {
			return nil, nil, err
		}
		aggs = append(aggs, agg)
		names = append(names, name)
	}
	return names, aggs, nil
}

func (b *Builder) compileAggAssignment(assignment dag.Assignment) (field.Path, *expr.Aggregator, error) {
	aggAST, ok := assignment.RHS.(*dag.Agg)
	if !ok {
		return nil, nil, errors.New("aggregator is not an aggregation expression")
	}
	lhs, err := compileLval(assignment.LHS)
	if err != nil {
		return nil, nil, fmt.Errorf("lhs of aggregation: %w", err)
	}
	m, err := b.compileAgg(aggAST)
	return lhs, m, err
}

func (b *Builder) compileAgg(agg *dag.Agg) (*expr.Aggregator, error) {
	name := agg.Name
	var err error
	var arg expr.Evaluator
	if agg.Expr != nil {
		arg, err = b.compileExpr(agg.Expr)
		if err != nil {
			return nil, err
		}
	}
	var where expr.Evaluator
	if agg.Where != nil {
		where, err = b.compileExpr(agg.Where)
		if err != nil {
			return nil, err
		}
	}
	return expr.NewAggregator(name, arg, where)
}
