package compile

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	recfield "github.com/brimsec/zq/field"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/reducer/field"
)

var (
	ErrUnknownField  = errors.New("unknown field")
	ErrFieldRequired = errors.New("field parameter required")
)

type Reducer struct {
	Target         recfield.Static // The name of the field where results are stored.
	TargetResolver expr.Evaluator
	Instantiate    func() reducer.Interface
}

func Compile(assignment ast.Assignment) (Reducer, error) {
	reducerAST, ok := assignment.RHS.(*ast.Reducer)
	if !ok {
		return Reducer{}, errors.New("reducer is not a reducer expression")
	}
	reducerOp := reducerAST.Operator
	var err error
	var rhs expr.Evaluator
	if reducerAST.Expr != nil {
		rhs, err = expr.CompileExpr(reducerAST.Expr)
		if err != nil {
			return Reducer{}, err
		}
	} else if reducerOp != "count" {
		// Currently,tThe only reducer that supports operator without
		// a field is count().
		return Reducer{}, ErrFieldRequired
	}
	var lhs recfield.Static
	// If there is a reducer assignment, the LHS is non-nil and we
	// compile.  Otherwise, we infer an LHS top-level field name from
	// the name of reducer function.
	if assignment.LHS == nil {
		lhs = recfield.New(reducerOp)
	} else {
		lhs, err = expr.CompileLval(assignment.LHS)
		if err != nil {
			return Reducer{}, fmt.Errorf("lhs of reducer expression: %w", err)
		}
	}
	var inst func() reducer.Interface
	switch reducerAST.Operator {
	case "count":
		inst = func() reducer.Interface {
			return &reducer.Count{Resolver: rhs}
		}
	case "first":
		inst = func() reducer.Interface {
			return &reducer.First{Resolver: rhs}
		}
	case "last":
		inst = func() reducer.Interface {
			return &reducer.Last{Resolver: rhs}
		}
	case "avg":
		inst = func() reducer.Interface {
			return &reducer.Avg{Resolver: rhs}
		}
	case "countdistinct":
		inst = func() reducer.Interface {
			return reducer.NewCountDistinct(rhs)
		}
	case "sum", "min", "max":
		inst = func() reducer.Interface {
			return &field.FieldReducer{Op: reducerOp, Resolver: rhs}
		}
	default:
		return Reducer{}, fmt.Errorf("unknown reducer op: %s", reducerOp)
	}
	return Reducer{
		Target:         lhs,
		TargetResolver: expr.NewDotExpr(lhs),
		Instantiate:    inst,
	}, nil
}
