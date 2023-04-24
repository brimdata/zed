package ast

import (
	"encoding/json"
	"errors"
	"fmt"

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
	ConstDecl{},
	Cut{},
	astzed.DefValue{},
	Drop{},
	Explode{},
	astzed.Enum{},
	astzed.Error{},
	Field{},
	File{},
	From{},
	FuncDecl{},
	Fuse{},
	Summarize{},
	Grep{},
	Head{},
	HTTP{},
	ID{},
	astzed.ImpliedValue{},
	Join{},
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
	astzed.Record{},
	Agg{},
	Regexp{},
	Glob{},
	RecordExpr{},
	Rename{},
	Scope{},
	Search{},
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
	VectorValue{},
	Where{},
	Yield{},
	Sample{},
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

func UnpackJSONAsSeq(buf []byte) (Seq, error) {
	var seq Seq
	if err := unpacker.UnmarshalInto(buf, &seq); err != nil {
		return nil, err
	}
	return seq, nil
}

func UnpackAsSeq(anon interface{}) (Seq, error) {
	body, err := json.Marshal(anon)
	if err != nil {
		return nil, fmt.Errorf("system error: ast.UnpackAsSeq: %w", err)
	}
	return UnpackJSONAsSeq(body)
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

func CopySeq(in Seq) Seq {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	out, err := UnpackJSONAsSeq(b)
	if err != nil {
		panic(err)
	}
	return out
}
