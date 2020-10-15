package reducer

//XXX in new model, need to do a semantic check on the reducers since they
// are compiled at runtime and you don't want to run a long time then catch
// the error that could have been caught earlier

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/anymath"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var (
	ErrBadValue = errors.New("bad value")
)

type Maker func() Interface

type Interface interface {
	Consume(*zng.Record)
	Result() zng.Value
}

type Decomposable interface {
	Interface
	ConsumePart(zng.Value) error
	ResultPart(*resolver.Context) (zng.Value, error)
}

type Stats struct {
	TypeMismatch  int64
	FieldNotFound int64
}

type Reducer struct {
	Stats
}

var (
	ErrUnknownField  = errors.New("unknown field")
	ErrFieldRequired = errors.New("field parameter required")
)

func NewMaker(op string, arg expr.Evaluator) (Maker, error) {
	if arg == nil && op != "count" {
		// Count is the only reducer that doesn't require an operator.
		return nil, ErrFieldRequired
	}
	switch op {
	case "count":
		return func() Interface {
			return &Count{arg: arg}
		}, nil
	case "first":
		return func() Interface {
			return &First{arg: arg}
		}, nil
	case "last":
		return func() Interface {
			return &Last{arg: arg}
		}, nil
	case "avg":
		return func() Interface {
			return &Avg{arg: arg}
		}, nil
	case "countdistinct":
		return func() Interface {
			return NewCountDistinct(arg)
		}, nil
	case "sum":
		return func() Interface {
			return newMathReducer(anymath.Add, arg)
		}, nil
	case "min":
		return func() Interface {
			return newMathReducer(anymath.Min, arg)
		}, nil
	case "max":
		return func() Interface {
			return newMathReducer(anymath.Max, arg)
		}, nil
	default:
		return nil, fmt.Errorf("unknown reducer op: %s", op)
	}
}
