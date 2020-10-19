package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Logical struct {
	Reducer
	and bool
	arg expr.Evaluator
	val bool
}

func (l *Logical) Consume(r *zng.Record) {
	if l.filter(r) {
		return
	}
	v, err := l.arg.Eval(r)
	if err != nil || v.IsNil() {
		return
	}
	if v.Type != zng.TypeBool {
		l.TypeMismatch++
		return
	}
	l.update(zng.IsTrue(v.Bytes))
}

func (l *Logical) update(val bool) {
	if l.and {
		l.val = l.val && val
	} else {
		l.val = l.val || val
	}
}

func (l *Logical) Result() zng.Value {
	if l.val {
		return zng.True
	}
	return zng.False
}

func (l *Logical) ConsumePart(p zng.Value) error {
	// DecodeBool returns error for nil bytes so we don't check it here.
	b, err := zng.DecodeBool(p.Bytes)
	if err != nil {
		return err
	}
	l.update(b)
	return nil
}

func (l *Logical) ResultPart(*resolver.Context) (zng.Value, error) {
	return l.Result(), nil
}
