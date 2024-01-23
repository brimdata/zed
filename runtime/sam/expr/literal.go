package expr

import "github.com/brimdata/zed"

type Literal struct {
	val zed.Value
}

var _ Evaluator = (*Literal)(nil)

func NewLiteral(val zed.Value) *Literal {
	return &Literal{val: val}
}

func (l Literal) Eval(Context, zed.Value) zed.Value {
	return l.val
}
