package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

type Literal struct {
	arena *zed.Arena
	val   zed.Value
}

var _ Evaluator = (*Literal)(nil)

func NewLiteral(arena *zed.Arena, val zed.Value) *Literal {
	return &Literal{arena, val}
}

func (l Literal) Eval(val vector.Any) vector.Any {
	return vector.NewConst(l.arena, l.val, val.Len(), nil)
}
