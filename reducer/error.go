package reducer

import (
	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
)

type Error struct {
	Reducer
	msg string
}

func NewError(def ast.Reducer, rec *zng.Record) *Error {
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

func (e *Error) Consume(t *zng.Record) {}

func (e *Error) Result() zeek.Value {
	return zeek.NewString(e.msg)
}
