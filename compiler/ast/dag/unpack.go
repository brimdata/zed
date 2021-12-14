package dag

import (
	"errors"

	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = unpack.New(
	astzed.Array{},
	ArrayExpr{},
	Assignment{},
	BinaryExpr{},
	Call{},
	Cast{},
	astzed.CastValue{},
	Conditional{},
	Const{},
	Cut{},
	astzed.DefValue{},
	Dot{},
	Drop{},
	Explode{},
	astzed.Enum{},
	File{},
	Filter{},
	From{},
	Fuse{},
	Summarize{},
	Head{},
	HTTP{},
	astzed.ImpliedValue{},
	Join{},
	astzed.Map{},
	MapExpr{},
	Shape{},
	Over{},
	Parallel{},
	Pass{},
	Path{},
	Pick{},
	Pool{},
	astzed.Primitive{},
	Put{},
	astzed.Record{},
	Agg{},
	RegexpMatch{},
	RegexpSearch{},
	RecordExpr{},
	Ref{},
	Rename{},
	Search{},
	SelectExpr{},
	SeqExpr{},
	Sequential{},
	astzed.Set{},
	SetExpr{},
	Sort{},
	Switch{},
	Tail{},
	Top{},
	Trunk{},
	astzed.TypeArray{},
	astzed.TypeDef{},
	astzed.TypeEnum{},
	astzed.TypeMap{},
	astzed.TypeName{},
	astzed.TypeNull{},
	astzed.TypePrimitive{},
	TypeProc{},
	astzed.TypeRecord{},
	astzed.TypeSet{},
	astzed.TypeUnion{},
	astzed.TypeValue{},
	UnaryExpr{},
	Uniq{},
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
