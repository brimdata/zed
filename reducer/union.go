package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Union struct {
	Reducer
	zctx *resolver.Context
	arg  expr.Evaluator
	typ  zng.Type
	set  map[string]struct{}
}

func newUnion(zctx *resolver.Context, arg, where expr.Evaluator) *Union {
	return &Union{
		Reducer: Reducer{where: where},
		zctx:    zctx,
		arg:     arg,
		set:     make(map[string]struct{}),
	}
}

func (u *Union) Consume(r *zng.Record) {
	if u.filter(r) {
		return
	}
	v, err := u.arg.Eval(r)
	if err != nil || v.IsNil() {
		return
	}
	if u.typ == nil {
		u.typ = v.Type
	} else if u.typ != v.Type {
		u.TypeMismatch++
		return
	}
	u.set[string(v.Bytes)] = struct{}{}
}

func (u *Union) Result() zng.Value {
	b := zcode.NewBuilder()
	container := zng.IsContainerType(u.typ)
	for s, _ := range u.set {
		if container {
			b.AppendContainer([]byte(s))
		} else {
			b.AppendPrimitive([]byte(s))
		}
	}
	setType := u.zctx.LookupTypeSet(u.typ)
	return zng.Value{setType, zng.NormalizeSet(b.Bytes())}
}

func (u *Union) ConsumePart(zv zng.Value) error {
	for it := zv.Iter(); !it.Done(); {
		elem, _, err := it.NextTagAndBody()
		if err != nil {
			return err
		}
		u.set[string(elem)] = struct{}{}
	}
	return nil
}

func (u *Union) ResultPart(*resolver.Context) (zng.Value, error) {
	return u.Result(), nil
}
