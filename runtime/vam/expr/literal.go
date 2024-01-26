package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

type Literal struct {
	val zed.Value
}

var _ Evaluator = (*Literal)(nil)

func NewLiteral(val zed.Value) *Literal {
	return &Literal{val: val}
}

func (l Literal) Eval(val vector.Any) vector.Any {
	return vector.NewConst(l.val, val.Len(), nil)
}
