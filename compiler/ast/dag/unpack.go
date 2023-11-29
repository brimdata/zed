package dag

import (
	"fmt"

	"github.com/brimdata/zed/pkg/unpack"
)

var unpacker = unpack.New(
	Agg{},
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
	Head{},
	HTTPScan{},
	Join{},
	Lister{},
	Literal{},
	Load{},
	MapCall{},
	MapExpr{},
	Merge{},
	Over{},
	OverExpr{},
	Pass{},
	PoolScan{},
	Put{},
	RecordExpr{},
	RegexpMatch{},
	RegexpSearch{},
	Rename{},
	Scatter{},
	Scope{},
	Search{},
	SeqScan{},
	SetExpr{},
	Shape{},
	Slicer{},
	Sort{},
	Spread{},
	Summarize{},
	Switch{},
	Tail{},
	This{},
	Top{},
	UnaryExpr{},
	Uniq{},
	Var{},
	Vectorize{},
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
