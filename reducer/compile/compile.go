package compile

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/reducer/field"
)

var (
	ErrUnknownField  = errors.New("unknown field")
	ErrFieldRequired = errors.New("field parameter required")
)

type CompiledReducer interface {
	Target() string // The name of the field where results are stored.
	TargetResolver() expr.FieldExprResolver
	Instantiate() reducer.Interface
}

func Compile(params ast.Reducer) (CompiledReducer, error) {
	name := params.Var
	var fld expr.FieldExprResolver
	if params.Field != nil {
		var err error
		if fld, err = expr.CompileFieldExpr(params.Field); err != nil {
			return nil, err
		}
	}
	target := expr.CompileFieldAccess(name)

	switch params.Op {
	case "Count":
		return reducer.NewCountProto(name, target, fld), nil
	case "First":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewFirstProto(name, target, fld), nil
	case "Last":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewLastProto(name, target, fld), nil
	case "Avg":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewAvgProto(name, target, fld), nil
	case "CountDistinct":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewCountDistinctProto(name, target, fld), nil
	case "Sum", "Min", "Max":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return field.NewFieldProto(name, target, fld, params.Op), nil
	default:
		return nil, fmt.Errorf("unknown reducer op: %s", params.Op)
	}
}
