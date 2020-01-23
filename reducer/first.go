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

func (fp *FirstProto) Instantiate(recType *zng.TypeRecord) Interface {
	typ, ok := recType.TypeOfField(fp.field)
	if !ok {
		typ = zng.TypeNull
	}
	return &First{Field: fp.field, typ: typ}
}

func NewFirstProto(target, field string) *FirstProto {
	return &FirstProto{target, field}
}

type First struct {
	Reducer
	Field  string
	typ    zng.Type
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
		return zng.Value{f.typ, nil}
	}
	v, _ := t.ValueByField(f.Field)
	return v
}
