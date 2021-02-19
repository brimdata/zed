package kernel

import (
	"fmt"

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
	aggLvals, aggFuncs, err := compileAggAssignments(p.Aggs, pctx.TypeContext, scope)
	if err != nil {
		return nil, err
	}
	sortdir := 1
	if p.InputDesc {
		sortdir = -1
	}
	return groupby.New(pctx, parent, keys, aggLvals, aggFuncs, p.Limit, sortdir, p.PartialsIn, p.PartialsIn)
}

func compileAggAssignments(assignments []AggAssignment, zctx *resolver.Context, scope *Scope) ([]field.Static, []*expr.Aggregator, error) {
	lvals := make([]field.Static, 0, len(assignments))
	aggs := make([]*expr.Aggregator, 0, len(assignments))
	for _, assignment := range assignments {
		lval, agg, err := compileAggFunc(zctx, scope, assignment)
		if err != nil {
			return nil, nil, err
		}
		aggs = append(aggs, agg)
		lvals = append(lvals, lval)
	}
	return lvals, aggs, nil
}

func compileAggFunc(zctx *resolver.Context, scope *Scope, assignment AggAssignment) (field.Static, *expr.Aggregator, error) {
	aggFunc := assignment.RHS
	var arg expr.Evaluator
	var err error
	if aggFunc.Arg != nil {
		arg, err = compileExpr(zctx, nil, aggFunc.Arg)
		if err != nil {
			return nil, nil, err
		}
	}
	// If there is a reducer assignment, the LHS is non-nil and we
	// compile.  Otherwise, we infer an LHS top-level field name from
	// the name of reducer function.
	var lhs field.Static
	if assignment.LHS == nil {
		lhs = field.New(aggFunc.Name)
	} else {
		lhs, err = CompileLval(assignment.LHS)
		if err != nil {
			return nil, nil, fmt.Errorf("lhs of aggregation: %w", err)
		}
	}
	var where expr.Filter
	if aggFunc.Where != nil {
		where, err = compileFilter(zctx, scope, aggFunc.Where)
		if err != nil {
			return nil, nil, err
		}
	}
	m, err := expr.NewAggregator(aggFunc.Name, arg, where)
	return lhs, m, err
}
