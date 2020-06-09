package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
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
	record   *zng.Record
}

func (l *Last) Consume(r *zng.Record) {
	if v := l.Resolver(r); v.Type == nil {
		return
	}
	l.record = r
}

func (l *Last) Result() zng.Value {
	r := l.record
	if r == nil {
		return zng.Value{Type: zng.TypeNull, Bytes: nil}
	}
	return l.Resolver(r)
}
