package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type mapCall struct {
	builder zcode.Builder
	eval    Evaluator
	inner   Evaluator
	zctx    *zed.Context

	// vals is used to reduce allocations
	vals []zed.Value
	// types is used to reduce allocations
	types []zed.Type
}

func NewMapCall(zctx *zed.Context, e, inner Evaluator) Evaluator {
	return &mapCall{eval: e, inner: inner, zctx: zctx}
}

func (a *mapCall) Eval(ectx Context, in *zed.Value) *zed.Value {
	val := a.eval.Eval(ectx, in)
	if val.IsError() {
		return val
	}
	elems, err := val.Elements()
	if err != nil {
		return ectx.CopyValue(*a.zctx.WrapError(err.Error(), in))
	}
	if len(elems) == 0 {
		return val
	}
	a.vals = a.vals[:0]
	a.types = a.types[:0]
	for _, elem := range elems {
		val := a.inner.Eval(ectx, &elem)
		a.vals = append(a.vals, *val)
		a.types = append(a.types, val.Type())
	}
	inner := a.innerType(a.types)
	bytes := a.buildVal(inner, a.vals)
	if _, ok := zed.TypeUnder(val.Type()).(*zed.TypeSet); ok {
		return ectx.NewValue(a.zctx.LookupTypeSet(inner), zed.NormalizeSet(bytes))
	}
	return ectx.NewValue(a.zctx.LookupTypeArray(inner), bytes)
}

func (a *mapCall) buildVal(inner zed.Type, vals []zed.Value) []byte {
	a.builder.Reset()
	if union, ok := inner.(*zed.TypeUnion); ok {
		for _, val := range a.vals {
			zed.BuildUnion(&a.builder, union.TagOf(val.Type()), val.Bytes())
		}
	} else {
		for _, val := range a.vals {
			a.builder.Append(val.Bytes())
		}
	}
	return a.builder.Bytes()
}

func (a *mapCall) innerType(types []zed.Type) zed.Type {
	types = zed.UniqueTypes(types)
	if len(types) == 1 {
		return types[0]
	}
	return a.zctx.LookupTypeUnion(types)
}
