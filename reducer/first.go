package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

type First struct {
	Reducer
	arg expr.Evaluator
	val *zng.Value
}

func (f *First) Consume(r *zng.Record) {
	if f.val != nil || f.filter(r) {
		return
	}
	v, err := f.arg.Eval(r)
	if err != nil || v.Type == nil {
		return
	}
	f.val = &v
}

func (f *First) Result() zng.Value {
	if f.val == nil {
		return zng.Value{Type: zng.TypeNull, Bytes: nil}
	}
	return *f.val
}
