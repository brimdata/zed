package reducer

import (
	"github.com/mccanne/zq/zng"
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
	record *zng.Record
}

func (f *First) Consume(r *zng.Record) {
	if f.record != nil {
		return
	}
	if _, ok := r.ColumnOfField(f.Field); !ok {
		return
	}
	f.record = r
}

func (f *First) Result() zng.Value {
	t := f.record
	if t == nil {
		return zng.Value{}
	}
	v, _ := t.ValueByField(f.Field)
	return v
}
