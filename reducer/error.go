package reducer

import (
	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type Error struct {
	Reducer
	msg string
}

func NewError(def ast.Reducer, rec *zbuf.Record) *Error {
	v := rec.ValueByField(def.Field)
	var msg string
	if v == nil {
		msg = def.Field + " not found"
	} else {
		msg = def.Op + " applied to type " + v.Type().String()
	}
	return &Error{
		msg: msg,
	}
}

func (e *Error) Consume(t *zbuf.Record) {}

func (e *Error) Result() zng.Value {
	return zng.NewString(e.msg)
}
