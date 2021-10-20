package expr

import "github.com/brimdata/zed"

type Literal struct {
	zv zed.Value
}

func NewLiteral(zv zed.Value) *Literal {
	return &Literal{zv}
}

func (l *Literal) Eval(*zed.Value) (zed.Value, error) {
	return l.zv, nil
}
