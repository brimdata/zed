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
	ErrBadValue      = errors.New("bad value")
	ErrFieldRequired = errors.New("field parameter required")
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
	where expr.Evaluator
}

func (r *Reducer) filter(rec *zng.Record) bool {
	if r.where == nil {
		return false
	}
	zv, err := r.where.Eval(rec)
	if err != nil {
		return true
	}
	return !zng.IsTrue(zv.Bytes)
}

func NewMaker(op string, arg, where expr.Evaluator) (Maker, error) {
	if arg == nil && op != "count" {
		// Count is the only reducer that doesn't require an operator.
		return nil, ErrFieldRequired
	}
	r := Reducer{where: where}
	switch op {
	case "count":
		return func() Interface {
			return &Count{Reducer: r, arg: arg}
		}, nil
	case "first":
		return func() Interface {
			return &First{Reducer: r, arg: arg}
		}, nil
	case "last":
		return func() Interface {
			return &Last{Reducer: r, arg: arg}
		}, nil
	case "avg":
		return func() Interface {
			return &Avg{Reducer: r, arg: arg}
		}, nil
	case "countdistinct":
		return func() Interface {
			return NewCountDistinct(arg, where)
		}, nil
	case "sum":
		return func() Interface {
			return newMathReducer(anymath.Add, arg, where)
		}, nil
	case "min":
		return func() Interface {
			return newMathReducer(anymath.Min, arg, where)
		}, nil
	case "max":
		return func() Interface {
			return newMathReducer(anymath.Max, arg, where)
		}, nil
	default:
		return nil, fmt.Errorf("unknown reducer op: %s", op)
	}
}
