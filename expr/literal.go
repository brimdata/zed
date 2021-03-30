package expr

import (
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Literal struct {
	zv zng.Value
}

//XXX This only works for primitive... will need zctx for complex literals.
// See issue #2335.
func NewLiteral(val ast.Primitive) (*Literal, error) {
	zv, err := zson.ParsePrimitive(val)
	if err != nil {
		return nil, err
	}
	return &Literal{zv}, nil
}

func NewLiteralVal(zv zng.Value) *Literal {
	return &Literal{zv}
}

func (l *Literal) Eval(*zng.Record) (zng.Value, error) {
	return l.zv, nil
}
