package ast

import (
	"errors"

	"github.com/brimsec/zq/pkg/unpack"
)

var unpacker = unpack.New().Init(
	SequentialProc{},
	ParallelProc{},
	SwitchProc{},
	SortProc{},
	CutProc{},
	PickProc{},
	DropProc{},
	HeadProc{},
	TailProc{},
	FilterProc{},
	FunctionCall{},
	PutProc{},
	RenameProc{},
	FuseProc{},
	UniqProc{},
	GroupByProc{},
	TopProc{},
	PassProc{},
	JoinProc{},
	ConstProc{},
	TypeProc{},
	Reducer{},
	Literal{},
	Identifier{},
	RootRecord{},
	TypeExpr{},
	Empty{},
	TypeNull{},
	TypeName{},
	TypeDef{},
	TypePrimitive{},
	TypeRecord{},
	TypeUnion{},
	TypeArray{},
	TypeSet{},
	TypeMap{},
	Search{},
	Assignment{},
	ImpliedValue{},
	DefValue{},
	CastValue{},
	Primitive{},
	Record{},
	Array{},
	Set{},
	Enum{},
	Map{},
	Entry{},
	TypeValue{},
	TypePrimitive{},
	TypeRecord{},
	TypeField{},
	TypeArray{},
	TypeSet{},
	TypeUnion{},
	TypeEnum{},
	TypeMap{},
	TypeNull{},
	TypeName{},
	TypeDef{},
).AddAs(BinaryExpression{}, "BinaryExpr").AddAs(SelectExpression{}, "SelectExpr").AddAs(UnaryExpression{}, "UnaryExpr").AddAs(CastExpression{}, "CastExpr").AddAs(ConditionalExpression{}, "ConditionalExpr")

// UnpackJSON transforms a JSON representation of a proc into an ast.Proc.
func UnpackJSON(buf []byte) (Proc, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	result, err := unpacker.UnpackBytes("op", buf)
	if err != nil {
		return nil, err
	}
	proc, ok := result.(Proc)
	if !ok {
		return nil, errors.New("JSON object is not a proc")
	}
	return proc, nil
}

func UnpackMapAsProc(m interface{}) (Proc, error) {
	object, err := unpacker.UnpackMap("op", m)
	if err != nil {
		return nil, err
	}
	proc, ok := object.(Proc)
	if !ok {
		return nil, errors.New("ast.UnpackMapAsProc: not a proc")
	}
	return proc, nil
}

func UnpackMapAsExpr(m interface{}) (Expression, error) {
	object, err := unpacker.UnpackMap("op", m)
	if err != nil {
		return nil, err
	}
	e, ok := object.(Expression)
	if !ok {
		return nil, errors.New("ast.UnpackMapAsExpr: not an expression")
	}
	return e, nil
}
