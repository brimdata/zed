package expr

import (
	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zson"
)

type Literal struct {
	zv zng.Value
}

//XXX This only works for primitive... will need zctx for complex literals.
// See issue #2335.
func NewLiteral(val ast.Literal) (*Literal, error) {
	zv, err := zson.ParsePrimitive(val.Type, val.Value)
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
