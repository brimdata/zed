package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Last struct {
	Reducer
	Resolver expr.Evaluator
	val      *zng.Value
}

func (l *Last) Consume(r *zng.Record) {
	v, err := l.Resolver.Eval(r)
	if err != nil || v.Type == nil {
		return
	}
	l.val = &v
}

func (l *Last) ConsumePart(p zng.Value) error {
	l.val = &p
	return nil
}

func (l *Last) Result() zng.Value {
	v := l.val
	if v == nil {
		return zng.Value{Type: zng.TypeNull, Bytes: nil}
	}
	return *v
}

func (l *Last) ResultPart(*resolver.Context) (zng.Value, error) {
	return l.Result(), nil
}
