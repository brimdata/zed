package reducer

import (
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/zng"
)

type FirstProto struct {
	target   string
	resolver expr.FieldExprResolver
}

func (fp *FirstProto) Target() string {
	return fp.target
}

func (fp *FirstProto) Instantiate(rec *zng.Record) Interface {
	v := fp.resolver(rec)
	if v.Type == nil {
		v.Type = zng.TypeNull
	}
	return &First{Resolver: fp.resolver, typ: v.Type}
}

func NewFirstProto(target string, field expr.FieldExprResolver) *FirstProto {
	return &FirstProto{target, field}
}

type First struct {
	Reducer
	Resolver expr.FieldExprResolver
	typ      zng.Type
	record   *zng.Record
}

func (f *First) Consume(r *zng.Record) {
	if f.record != nil {
		return
	}
	if v := f.Resolver(r); v.Type == nil {
		return
	}
	f.record = r
}

func (f *First) Result() zng.Value {
	if f.record == nil {
		return zng.Value{Type: f.typ, Bytes: nil}
	}
	return f.Resolver(f.record)
}
