package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type apply struct {
	builder zcode.Builder
	eval    Evaluator
	fn      Function
	zctx    *zed.Context

	// vals is used to reduce allocations
	vals []zed.Value
	// types is used to reduce allocations
	types []zed.Type
}

func NewApplyFunc(zctx *zed.Context, e Evaluator, fn Function) Evaluator {
	return &apply{eval: e, fn: fn, zctx: zctx}
}

func (a *apply) Eval(ectx Context, in *zed.Value) *zed.Value {
	v := a.eval.Eval(ectx, in)
	if v.IsError() {
		return v
	}
	elems, err := v.Elements()
	if err != nil {
		return ectx.CopyValue(*a.zctx.WrapError(err.Error(), in))
	}
	if len(elems) == 0 {
		return v
	}
	a.vals = a.vals[:0]
	a.types = a.types[:0]
	for _, elem := range elems {
		out := a.fn.Call(ectx, []zed.Value{elem})
		a.vals = append(a.vals, *out)
		a.types = append(a.types, out.Type)
	}
	inner := a.innerType(a.types)
	a.builder.Reset()
	if union, ok := inner.(*zed.TypeUnion); ok {
		for _, val := range a.vals {
			zed.BuildUnion(&a.builder, union.TagOf(val.Type), val.Bytes())
		}
	} else {
		for _, val := range a.vals {
			a.builder.Append(val.Bytes())
		}
	}
	if _, ok := zed.TypeUnder(in.Type).(*zed.TypeSet); ok {
		return ectx.NewValue(a.zctx.LookupTypeSet(inner), zed.NormalizeSet(a.builder.Bytes()))
	}
	return ectx.NewValue(a.zctx.LookupTypeArray(inner), a.builder.Bytes())
}

func (a *apply) innerType(types []zed.Type) zed.Type {
	types = zed.UniqueTypes(types)
	if len(types) == 1 {
		return types[0]
	}
	return a.zctx.LookupTypeUnion(types)
}
