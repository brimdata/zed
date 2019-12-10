package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type FirstProto struct {
	target string
	field  string
}

func (fp *FirstProto) Target() string {
	return fp.target
}

func (fp *FirstProto) Instantiate() Interface {
	return &First{Field: fp.field}
}

func NewFirstProto(target, field string) *FirstProto {
	return &FirstProto{target, field}
}

type First struct {
	Reducer
	Field  string
	record *zson.Record
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
		return &zeek.Unset{}
	}
	return t.ValueByField(f.Field)
}
