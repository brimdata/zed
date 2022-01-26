package dag

import (
	"errors"

	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = unpack.New(
	ArrayExpr{},
	Assignment{},
	BinaryExpr{},
	Call{},
	Cast{},
	Conditional{},
	Cut{},
	Dot{},
	Drop{},
	Explode{},
	File{},
	Filter{},
	From{},
	Fuse{},
	Summarize{},
	Head{},
	HTTP{},
	Join{},
	Literal{},
	MapExpr{},
	Shape{},
	Over{},
	Parallel{},
	Pass{},
	Pick{},
	Pool{},
	Put{},
	Agg{},
	RegexpMatch{},
	RegexpSearch{},
	RecordExpr{},
	Rename{},
	Let{},
	Search{},
	Sequential{},
	SetExpr{},
	Sort{},
	Switch{},
	Tail{},
	This{},
	Top{},
	Trunk{},
	UnaryExpr{},
	Uniq{},
	Var{},
	Yield{},
)

func UnpackJSON(buf []byte) (interface{}, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	return unpacker.Unmarshal(buf)
}

// UnpackJSONAsOp transforms a JSON representation of an operator into an dag.Op.
func UnpackJSONAsOp(buf []byte) (Op, error) {
	result, err := UnpackJSON(buf)
	if result == nil || err != nil {
		return nil, err
	}
	op, ok := result.(Op)
	if !ok {
		return nil, errors.New("JSON object is not a DAG operator")
	}
	return op, nil
}

func UnpackMapAsOp(m interface{}) (Op, error) {
	object, err := unpacker.UnmarshalObject(m)
	if object == nil || err != nil {
		return nil, err
	}
	op, ok := object.(Op)
	if !ok {
		return nil, errors.New("dag.UnpackMapAsOp: not an Op")
	}
	return op, nil
}

func UnpackMapAsExpr(m interface{}) (Expr, error) {
	object, err := unpacker.UnmarshalObject(m)
	if object == nil || err != nil {
		return nil, err
	}
	e, ok := object.(Expr)
	if !ok {
		return nil, errors.New("dag.UnpackMapAsExpr: not an Expr")
	}
	return e, nil
}
