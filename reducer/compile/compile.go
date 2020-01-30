package compile

import (
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/reducer/field"
	"github.com/mccanne/zq/zng"
)

var (
	ErrUnknownField  = errors.New("unknown field")
	ErrFieldRequired = errors.New("field parameter required")
)

type CompiledReducer interface {
	Target() string // The name of the field where results are stored.
	Instantiate(*zng.Record) reducer.Interface
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

	switch params.Op {
	case "Count":
		return reducer.NewCountProto(name, fld), nil
	case "First":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewFirstProto(name, fld), nil
	case "Last":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewLastProto(name, fld), nil
	case "Avg":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewAvgProto(name, fld), nil
	case "CountDistinct":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return reducer.NewCountDistinctProto(name, fld), nil
	case "Sum", "Min", "Max":
		if fld == nil {
			return nil, ErrFieldRequired
		}
		return field.NewFieldProto(name, fld, params.Op), nil
	default:
		return nil, fmt.Errorf("unknown reducer op: %s", params.Op)
	}
}
