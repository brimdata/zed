package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

type Last struct {
	Reducer
	arg expr.Evaluator
	val *zng.Value
}

func (l *Last) Consume(r *zng.Record) {
	if l.filter(r) {
		return
	}
	v, err := l.arg.Eval(r)
	if err != nil || v.Type == nil {
		return
	}
	v = v.Copy()
	l.val = &v
}

func (l *Last) Result() zng.Value {
	v := l.val
	if v == nil {
		return zng.Value{Type: zng.TypeNull, Bytes: nil}
	}
	return *v
}
