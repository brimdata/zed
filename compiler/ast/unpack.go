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
	OpExpr{},
	BinaryExpr{},
	Call{},
	Cast{},
	astzed.CastValue{},
	Conditional{},
	Cut{},
	astzed.DefValue{},
	Drop{},
	Explode{},
	astzed.Enum{},
	astzed.Error{},
	Field{},
	File{},
	From{},
	Fuse{},
	Summarize{},
	Grep{},
	Head{},
	HTTP{},
	ID{},
	astzed.ImpliedValue{},
	Join{},
	Layout{},
	Let{},
	Merge{},
	Over{},
	Trunk{},
	astzed.Map{},
	MapExpr{},
	Shape{},
	OverExpr{},
	Parallel{},
	Pass{},
	Pool{},
	astzed.Primitive{},
	Put{},
	Range{},
	astzed.Record{},
	Agg{},
	Regexp{},
	Glob{},
	RecordExpr{},
	Rename{},
	Search{},
	Sequential{},
	astzed.Set{},
	SetExpr{},
	Spread{},
	SQLExpr{},
	SQLOrderBy{},
	Sort{},
	String{},
	Switch{},
	Tail{},
	Term{},
	Top{},
	astzed.TypeArray{},
	astzed.TypeDef{},
	astzed.TypeEnum{},
	astzed.TypeError{},
	astzed.TypeMap{},
	astzed.TypeName{},
	astzed.TypeNull{},
	astzed.TypePrimitive{},
	astzed.TypeRecord{},
	astzed.TypeSet{},
	astzed.TypeUnion{},
	astzed.TypeValue{},
	UnaryExpr{},
	Uniq{},
	Where{},
	Yield{},
)

func UnpackJSON(buf []byte) (interface{}, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	return unpacker.Unmarshal(buf)
}

// UnpackJSONAsOp transforms a JSON representation of an operator into an Op.
func UnpackJSONAsOp(buf []byte) (Op, error) {
	result, err := UnpackJSON(buf)
	if result == nil || err != nil {
		return nil, err
	}
	o, ok := result.(Op)
	if !ok {
		return nil, errors.New("not an operator")
	}
	return o, nil
}

func UnpackMapAsOp(m interface{}) (Op, error) {
	object, err := unpacker.UnmarshalObject(m)
	if object == nil || err != nil {
		return nil, err
	}
	o, ok := object.(Op)
	if !ok {
		return nil, errors.New("not an operator")
	}
	return o, nil
}

func Copy(in Op) Op {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	out, err := UnpackJSONAsOp(b)
	if err != nil {
		panic(err)
	}
	return out
}
