package compile

import (
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/reducer/field"
)

var (
	ErrUnknownField  = errors.New("unknown field")
	ErrFieldRequired = errors.New("field parameter required")
)

type CompiledReducer interface {
	Target()      string  // The name of the field where results are stored.
	Instantiate() reducer.Interface
}

func Compile(params ast.Reducer) (CompiledReducer, error) {
	name := params.Var
	fld := params.Field

	switch params.Op {
	case "Count":
		return reducer.NewCountProto(name, fld), nil
	case "First":
		if fld == "" {
			return nil, ErrFieldRequired
		}
		return reducer.NewFirstProto(name, fld), nil
	case "Last":
		if fld == "" {
			return nil, ErrFieldRequired
		}
		return reducer.NewLastProto(name, fld), nil
	case "Avg":
		if fld == "" {
			return nil, ErrFieldRequired
		}
		return reducer.NewAvgProto(name, fld), nil
	case "CountDistinct":
		if fld == "" {
			return nil, ErrFieldRequired
		}
		return reducer.NewCountDistinctProto(name, fld), nil
	case "Sum", "Min", "Max":
		if fld == "" {
			return nil, ErrFieldRequired
		}
		return field.NewFieldProto(name, fld, params.Op), nil
	default:
		return nil, fmt.Errorf("unknown reducer op: %s", params.Op)
	}
}
