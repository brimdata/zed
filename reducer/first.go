package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type FirstProto struct {
	target              string
	resolver, tresolver expr.FieldExprResolver
}

func (fp *FirstProto) Target() string {
	return fp.target
}

func (fp *FirstProto) Instantiate() Interface {
	return &First{Resolver: fp.resolver}
}

func (fp *FirstProto) TargetResolver() expr.FieldExprResolver {
	return fp.tresolver
}

func NewFirstProto(target string, tresolver, resolver expr.FieldExprResolver) *FirstProto {
	return &FirstProto{
		target:    target,
		tresolver: tresolver,
		resolver:  resolver,
	}
}

type First struct {
	Reducer
	Resolver expr.FieldExprResolver
	val      *zng.Value
}

func (f *First) Consume(r *zng.Record) {
	if f.val != nil {
		return
	}
	v := f.Resolver(r)
	if v.Type == nil {
		return
	}
	f.val = &v
}

func (f *First) ConsumePart(p zng.Value) error {
	if f.val != nil || p.Type == zng.TypeNull {
		return nil
	}
	f.val = &p
	return nil
}

func (f *First) Result() zng.Value {
	if f.val == nil {
		return zng.Value{Type: zng.TypeNull, Bytes: nil}
	}
	return *f.val
}

func (f *First) ResultPart(*resolver.Context) (zng.Value, error) {
	return f.Result(), nil
}
