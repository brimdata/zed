package expr

import "github.com/brimdata/zed"

type Literal struct {
	arena *zed.Arena
	val   zed.Value
}

var _ Evaluator = (*Literal)(nil)

func NewLiteral(arena *zed.Arena, val zed.Value) *Literal {
	return &Literal{arena, val}
}

func (l Literal) Eval(Context, zed.Value) zed.Value {
	return l.val
}
