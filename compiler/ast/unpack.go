package ast

import (
	"errors"

	"github.com/brimsec/zq/pkg/unpack"
)

var unpacker = unpack.New(
	Array{},
	Assignment{},
	BinaryExpr{},
	Call{},
	Cast{},
	CastValue{},
	Conditional{},
	Const{},
	Cut{},
	DefValue{},
	Drop{},
	Enum{},
	Filter{},
	Fuse{},
	Summarize{},
	Head{},
	Id{},
	ImpliedValue{},
	Join{},
	Literal{},
	Map{},
	Shape{},
	Parallel{},
	Pass{},
	Path{},
	Pick{},
	Primitive{},
	Put{},
	Record{},
	Agg{},
	Ref{},
	Rename{},
	Root{},
	Search{},
	SelectExpr{},
	SeqExpr{},
	Sequential{},
	Set{},
	Sort{},
	Switch{},
	Tail{},
	Top{},
	TypeArray{},
	TypeDef{},
	TypeEnum{},
	TypeMap{},
	TypeName{},
	TypeNull{},
	TypePrimitive{},
	TypeProc{},
	TypeRecord{},
	TypeSet{},
	TypeUnion{},
	TypeValue{},
	UnaryExpr{},
	Uniq{},
)

func UnpackJSON(buf []byte) (interface{}, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	return unpacker.UnpackBytes(buf)
}

// UnpackJSONAsProc transforms a JSON representation of a proc into an ast.Proc.
func UnpackJSONAsProc(buf []byte) (Proc, error) {
	result, err := UnpackJSON(buf)
	if result == nil || err != nil {
		return nil, err
	}
	proc, ok := result.(Proc)
	if !ok {
		return nil, errors.New("JSON object is not a proc")
	}
	return proc, nil
}

func UnpackMapAsProc(m interface{}) (Proc, error) {
	object, err := unpacker.UnpackMap(m)
	if object == nil || err != nil {
		return nil, err
	}
	proc, ok := object.(Proc)
	if !ok {
		return nil, errors.New("ast.UnpackMapAsProc: not a proc")
	}
	return proc, nil
}

func UnpackMapAsExpr(m interface{}) (Expr, error) {
	object, err := unpacker.UnpackMap(m)
	if object == nil || err != nil {
		return nil, err
	}
	e, ok := object.(Expr)
	if !ok {
		return nil, errors.New("ast.UnpackMapAsExpr: not an expression")
	}
	return e, nil
}
