package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Last struct {
	Reducer
	Field  string
	record *zson.Record
}

func NewLast(name, field string) *Last {
	return &Last{
		Reducer: New(name),
		Field:   field,
	}
}

func (l *Last) Consume(r *zson.Record) {
	if _, ok := r.ColumnOfField(l.Field); !ok {
		return
	}
	l.record = r
}

func (l *Last) Result() zeek.Value {
	r := l.record
	if r == nil {
		return &zeek.None{}
	}
	return r.ValueByField(l.Field)
}
