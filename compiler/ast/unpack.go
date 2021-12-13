package ast

import (
	"encoding/json"
	"errors"

	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = unpack.New(
	astzed.Array{},
	ArrayExpr{},
	Assignment{},
	OpAssignment{},
	BinaryExpr{},
	Call{},
	Cast{},
	astzed.CastValue{},
	Conditional{},
	Const{},
	Cut{},
	astzed.DefValue{},
	Drop{},
	Explode{},
	astzed.Enum{},
	FieldCutter{},
	File{},
	Filter{},
	From{},
	Fuse{},
	Summarize{},
	Head{},
	HTTP{},
	ID{},
	astzed.ImpliedValue{},
	Join{},
	Layout{},
	Over{},
	Trunk{},
	astzed.Map{},
	MapExpr{},
	Shape{},
	TypeSplitter{},
	Parallel{},
	Pass{},
	Pick{},
	Pool{},
	astzed.Primitive{},
	Put{},
	Range{},
	astzed.Record{},
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
	astzed.Set{},
	SetExpr{},
	SQLExpr{},
	SQLOrderBy{},
	Sort{},
	Switch{},
	Tail{},
	Top{},
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
