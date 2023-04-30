package dag

import (
	"fmt"

	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = unpack.New(
	ArrayExpr{},
	Assignment{},
	BinaryExpr{},
	Call{},
	Combine{},
	Conditional{},
	Cut{},
	Deleter{},
	Dot{},
	Drop{},
	Explode{},
	Field{},
	FileScan{},
	Filter{},
	Fork{},
	Func{},
	Fuse{},
	Summarize{},
	Head{},
	HTTPScan{},
	Join{},
	Lister{},
	Literal{},
	MapExpr{},
	Merge{},
	Shape{},
	SeqScan{},
	Slicer{},
	Spread{},
	Over{},
	OverExpr{},
	Pass{},
	PoolScan{},
	Put{},
	Agg{},
	RegexpMatch{},
	RegexpSearch{},
	RecordExpr{},
	Rename{},
	Scatter{},
	Scope{},
	Search{},
	SetExpr{},
	Sort{},
	Switch{},
	Tail{},
	This{},
	Top{},
	UnaryExpr{},
	Uniq{},
	Var{},
	VectorValue{},
	Yield{},
)

// UnmarshalOp transforms a JSON representation of an operator into an Op.
func UnmarshalOp(buf []byte) (Op, error) {
	var op Op
	if err := unpacker.Unmarshal(buf, &op); err != nil {
		return nil, fmt.Errorf("internal error: JSON object is not a DAG operator: %w", err)
	}
	return op, nil
}
