package groupby

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zng/builder"
	"github.com/brimsec/zq/zng/resolver"
)

func compileParams(node *ast.GroupByProc, zctx *resolver.Context) (*Params, error) {
	keys := []Key{}
	var targets []field.Static
	for k, key := range node.Keys {
		name, rhs, err := expr.CompileAssignment(zctx, &key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, Key{
			tmp:  fmt.Sprintf("c%d", k),
			name: name,
			expr: rhs,
		})
		targets = append(targets, name)
	}
	reducerMakers := make([]reducerMaker, 0, len(node.Reducers))
	for _, assignment := range node.Reducers {
		name, f, err := CompileReducer(zctx, assignment)
		if err != nil {
			return nil, err
		}
		reducerMakers = append(reducerMakers, reducerMaker{name, f})
	}
	builder, err := builder.NewColumnBuilder(zctx, targets)
	if err != nil {
		return nil, fmt.Errorf("compiling groupby: %w", err)
	}
	if (node.ConsumePart || node.EmitPart) && !decomposable(reducerMakers) {
		return nil, errors.New("partial input or output requested with non-decomposable reducers")
	}
	return &Params{
		limit:        node.Limit,
		keys:         keys,
		makers:       reducerMakers,
		builder:      builder,
		inputSortDir: node.InputSortDir,
		consumePart:  node.ConsumePart,
		emitPart:     node.EmitPart,
	}, nil
}

func CompileReducer(zctx *resolver.Context, assignment ast.Assignment) (field.Static, reducer.Maker, error) {
	reducerAST, ok := assignment.RHS.(*ast.Reducer)
	if !ok {
		return nil, nil, errors.New("reducer is not a reducer expression")
	}
	reducerOp := reducerAST.Operator
	var err error
	var arg expr.Evaluator
	if reducerAST.Expr != nil {
		arg, err = expr.CompileExpr(zctx, reducerAST.Expr)
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
		lhs, err = expr.CompileLval(assignment.LHS)
		if err != nil {
			return nil, nil, fmt.Errorf("lhs of reducer expression: %w", err)
		}
	}
	var where expr.Evaluator
	if reducerAST.Where != nil {
		where, err = expr.CompileExpr(zctx, reducerAST.Where)
		if err != nil {
			return nil, nil, err
		}
	}
	f, err := reducer.NewMaker(reducerOp, arg, where)
	return lhs, f, err
}
