package reducer

import (
	"errors"

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
	val  map[string]struct{}
	size int
}

func newUnion(zctx *resolver.Context, arg, where expr.Evaluator) *Union {
	return &Union{
		Reducer: Reducer{where: where},
		zctx:    zctx,
		arg:     arg,
		val:     make(map[string]struct{}),
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
	u.update(v.Bytes)
}

func (u *Union) update(b zcode.Bytes) {
	if _, ok := u.val[string(b)]; !ok {
		u.val[string(b)] = struct{}{}
		u.size += len(b)
		for u.size > MaxValueSize {
			u.deleteOne()
			u.MemExceeded++
		}
	}
}

func (u *Union) deleteOne() {
	for key := range u.val {
		u.size -= len(key)
		delete(u.val, key)
		return
	}
}

func (u *Union) Result() zng.Value {
	var b zcode.Builder
	container := zng.IsContainerType(u.typ)
	for s, _ := range u.val {
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
	if u.typ == nil {
		typ, ok := zv.Type.(*zng.TypeSet)
		if !ok {
			return errors.New("partial not a set type")
		}
		u.typ = typ.Type
	}
	for it := zv.Iter(); !it.Done(); {
		elem, _, err := it.Next()
		if err != nil {
			return err
		}
		u.update(elem)
	}
	return nil
}

func (u *Union) ResultPart(*resolver.Context) (zng.Value, error) {
	return u.Result(), nil
}
