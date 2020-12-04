package compiler

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/groupby"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zng/resolver"
)

func compileGroupBy(pctx *proc.Context, parent proc.Interface, node *ast.GroupByProc) (*groupby.Proc, error) {
	keys, err := compileAssignments(node.Keys, pctx.TypeContext)
	if err != nil {
		return nil, err
	}
	names, reducers, err := compileReducers(node.Reducers, pctx.TypeContext)
	if err != nil {
		return nil, err
	}
	return groupby.New(pctx, parent, keys, names, reducers, node.Limit, node.InputSortDir, node.ConsumePart, node.EmitPart)
}

func compileReducers(assignments []ast.Assignment, zctx *resolver.Context) ([]field.Static, []reducer.Maker, error) {
	names := make([]field.Static, 0, len(assignments))
	reducers := make([]reducer.Maker, 0, len(assignments))
	for _, assignment := range assignments {
		name, maker, err := compileReducer(zctx, assignment)
		if err != nil {
			return nil, nil, err
		}
		reducers = append(reducers, maker)
		names = append(names, name)
	}
	return names, reducers, nil
}

func compileReducer(zctx *resolver.Context, assignment ast.Assignment) (field.Static, reducer.Maker, error) {
	reducerAST, ok := assignment.RHS.(*ast.Reducer)
	if !ok {
		return nil, nil, errors.New("reducer is not a reducer expression")
	}
	reducerOp := reducerAST.Operator
	var err error
	var arg expr.Evaluator
	if reducerAST.Expr != nil {
		arg, err = CompileExpr(zctx, reducerAST.Expr)
		if err != nil {
			return nil, nil, err
		}
	}
	// If there is a reducer assignment, the LHS is non-nil and we
	// compile.  Otherwise, we infer an LHS top-level field name from
	// the name of reducer function.
	var lhs field.Static
	if assignment.LHS == nil {
		lhs = field.New(reducerOp)
	} else {
		lhs, err = CompileLval(assignment.LHS)
		if err != nil {
			return nil, nil, fmt.Errorf("lhs of reducer expression: %w", err)
		}
	}
	var where expr.Evaluator
	if reducerAST.Where != nil {
		where, err = CompileExpr(zctx, reducerAST.Where)
		if err != nil {
			return nil, nil, err
		}
	}
	m, err := reducer.NewMaker(reducerOp, arg, where)
	return lhs, m, err
}

func IsDecomposable(assignments []ast.Assignment) bool {
	zctx := resolver.NewContext()
	for _, assignment := range assignments {
		_, maker, err := compileReducer(zctx, assignment)
		if err != nil {
			return false
		}
		if _, ok := maker(nil).(reducer.Decomposable); !ok {
			return false
		}
	}
	return true
}
