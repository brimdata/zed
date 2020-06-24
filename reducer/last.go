package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type LastProto struct {
	target   string
	resolver expr.FieldExprResolver
}

func (lp *LastProto) Target() string {
	return lp.target
}

func (lp *LastProto) Instantiate() Interface {
	return &Last{Resolver: lp.resolver}
}

func NewLastProto(target string, resolver expr.FieldExprResolver) *LastProto {
	return &LastProto{target, resolver}
}

type Last struct {
	Reducer
	Resolver expr.FieldExprResolver
	val      *zng.Value
}

func (l *Last) Consume(r *zng.Record) {
	v := l.Resolver(r)
	if v.Type == nil {
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
