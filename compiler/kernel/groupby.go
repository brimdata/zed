package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zq/compiler/ast"
	"github.com/brimdata/zq/expr"
	"github.com/brimdata/zq/field"
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/proc/groupby"
	"github.com/brimdata/zq/zng/resolver"
)

func compileGroupBy(pctx *proc.Context, scope *Scope, parent proc.Interface, node *ast.Summarize) (*groupby.Proc, error) {
	keys, err := compileAssignments(node.Keys, pctx.Zctx, scope)
	if err != nil {
		return nil, err
	}
	names, reducers, err := compileAggs(node.Aggs, scope, pctx.Zctx)
	if err != nil {
		return nil, err
	}
	return groupby.New(pctx, parent, keys, names, reducers, node.Limit, node.InputSortDir, node.PartialsIn, node.PartialsOut)
}

func compileAggs(assignments []ast.Assignment, scope *Scope, zctx *resolver.Context) ([]field.Static, []*expr.Aggregator, error) {
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
	aggAST, ok := assignment.RHS.(*ast.Agg)
	if !ok {
		return nil, nil, errors.New("aggregator is not an aggregation expression")
	}
	aggName := aggAST.Name
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
		lhs = field.New(aggName)
	} else {
		lhs, err = compileLval(assignment.LHS)
		if err != nil {
			return nil, nil, fmt.Errorf("lhs of aggregation: %w", err)
		}
	}
	var where expr.Filter
	if aggAST.Where != nil {
		where, err = CompileFilter(zctx, scope, aggAST.Where)
		if err != nil {
			return nil, nil, err
		}
	}
	m, err := expr.NewAggregator(aggName, arg, where)
	return lhs, m, err
}
