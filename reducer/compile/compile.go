package compile

import (
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/reducer/field"
)

var (
	ErrUnknownField  = errors.New("unknown field")
	ErrFieldRequired = errors.New("field parameter required")
)

func chkfield(field string) error {
	if field == "" {
		// XXX move this to semantic pass
		return ErrFieldRequired
	}
	return nil
}

func Compile(params ast.Reducer, rec *zson.Record) (reducer.Interface, error) {
	name := params.Var
	fld := params.Field
	switch params.Op {
	case "Count":
		return reducer.NewCount(name, fld), nil
	case "First":
		if err := chkfield(fld); err != nil {
			return nil, err
		}
		return reducer.NewFirst(name, fld), nil
	case "Last":
		if err := chkfield(fld); err != nil {
			return nil, err
		}
		return reducer.NewLast(name, fld), nil
	case "Avg":
		if err := chkfield(fld); err != nil {
			return nil, err
		}
		return reducer.NewAvg(name, fld), nil
	case "Sum", "Min", "Max":
		if err := chkfield(fld); err != nil {
			return nil, err
		}
		z := rec.ValueByField(fld)
		if z == nil {
			return nil, ErrUnknownField
		}
		switch z.Type().(type) {
		case *zeek.TypeOfInt:
			return field.NewInt(name, fld, params.Op), nil
		case *zeek.TypeOfCount:
			return field.NewCount(name, fld, params.Op), nil
		case *zeek.TypeOfDouble:
			return field.NewDouble(name, fld, params.Op), nil
		case *zeek.TypeOfInterval:
			return field.NewInterval(name, fld, params.Op), nil
		case *zeek.TypeOfTime:
			return field.NewTime(name, fld, params.Op), nil
		default:
			return nil, reducer.ErrUnsupportedType
		}
	default:
		return nil, fmt.Errorf("unknown reducer op: %s", params.Op)
	}
}
