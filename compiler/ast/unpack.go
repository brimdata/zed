package ast

import (
	"encoding/json"
	"errors"

	"github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = unpack.New(
	zed.Array{},
	ArrayExpr{},
	Assignment{},
	OpExprs{},
	BinaryExpr{},
	Call{},
	Cast{},
	zed.CastValue{},
	Conditional{},
	Const{},
	Cut{},
	zed.DefValue{},
	Drop{},
	Explode{},
	zed.Enum{},
	FieldCutter{},
	File{},
	Filter{},
	From{},
	Fuse{},
	Summarize{},
	Head{},
	HTTP{},
	ID{},
	zed.ImpliedValue{},
	Join{},
	Layout{},
	Trunk{},
	zed.Map{},
	MapExpr{},
	Shape{},
	TypeSplitter{},
	Parallel{},
	Pass{},
	Pick{},
	Pool{},
	zed.Primitive{},
	Put{},
	Range{},
	zed.Record{},
	Agg{},
	RegexpMatch{},
	RegexpSearch{},
	RecordExpr{},
	Rename{},
	Root{},
	Search{},
	SelectExpr{},
	SeqExpr{},
	Sequential{},
	zed.Set{},
	SetExpr{},
	SQLExpr{},
	SQLOrderBy{},
	Sort{},
	Switch{},
	Tail{},
	Top{},
	zed.TypeArray{},
	zed.TypeDef{},
	zed.TypeEnum{},
	zed.TypeMap{},
	zed.TypeName{},
	zed.TypeNull{},
	zed.TypePrimitive{},
	TypeProc{},
	zed.TypeRecord{},
	zed.TypeSet{},
	zed.TypeUnion{},
	zed.TypeValue{},
	UnaryExpr{},
	Uniq{},
)

func UnpackJSON(buf []byte) (interface{}, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	return unpacker.Unmarshal(buf)
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
	object, err := unpacker.UnmarshalObject(m)
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
	object, err := unpacker.UnmarshalObject(m)
	if object == nil || err != nil {
		return nil, err
	}
	e, ok := object.(Expr)
	if !ok {
		return nil, errors.New("ast.UnpackMapAsExpr: not an expression")
	}
	return e, nil
}

func Copy(in Proc) Proc {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	out, err := UnpackJSONAsProc(b)
	if err != nil {
		panic(err)
	}
	return out
}
