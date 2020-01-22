package reducer

import (
	"github.com/mccanne/zq/zng"
)

type LastProto struct {
	target string
	field  string
}

func (lp *LastProto) Target() string {
	return lp.target
}

func (lp *LastProto) Instantiate(recType *zng.TypeRecord) Interface {
	typ, ok := recType.TypeOfField(lp.field)
	if !ok {
		typ = zng.TypeNull
	}
	return &Last{Field: lp.field, typ: typ}
}

func NewLastProto(target, field string) *LastProto {
	return &LastProto{target, field}
}

type Last struct {
	Reducer
	Field  string
	typ    zng.Type
	record *zng.Record
}

func (l *Last) Consume(r *zng.Record) {
	if _, ok := r.ColumnOfField(l.Field); !ok {
		return
	}
	l.record = r
}

func (l *Last) Result() zng.Value {
	r := l.record
	if r == nil {
		return zng.Value{l.typ, nil}
	}
	v, _ := r.ValueByField(l.Field)
	return v
}
