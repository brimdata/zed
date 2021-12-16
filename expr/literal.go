package expr

import "github.com/brimdata/zed"

type Literal struct {
	zv zed.Value
}

var _ Evaluator = (*Literal)(nil)

func NewLiteral(zv zed.Value) *Literal {
	return &Literal{zv}
}

func (l *Literal) Eval(*zed.Value, *Scope) *zed.Value {
	return &l.zv, nil
}
