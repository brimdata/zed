package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type First struct {
	Reducer
	Field  string
	record *zson.Record
}

func NewFirst(name, field string) *First {
	return &First{
		Reducer: New(name),
		Field:   field,
	}
}

func (f *First) Consume(r *zson.Record) {
	if f.record != nil {
		return
	}
	if _, ok := r.ColumnOfField(f.Field); !ok {
		return
	}
	f.record = r
}

func (f *First) Result() zeek.Value {
	t := f.record
	if t == nil {
		return &zeek.None{}
	}
	return t.ValueByField(f.Field)
}
