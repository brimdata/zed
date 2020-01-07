package reducer

import (
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type LastProto struct {
	target string
	field  string
}

func (lp *LastProto) Target() string {
	return lp.target
}

func (lp *LastProto) Instantiate() Interface {
	return &Last{Field: lp.field}
}

func NewLastProto(target, field string) *LastProto {
	return &LastProto{target, field}
}

type Last struct {
	Reducer
	Field  string
	record *zbuf.Record
}

func (l *Last) Consume(r *zbuf.Record) {
	if _, ok := r.ColumnOfField(l.Field); !ok {
		return
	}
	l.record = r
}

func (l *Last) Result() zng.Value {
	r := l.record
	if r == nil {
		return &zng.Unset{}
	}
	return r.ValueByField(l.Field)
}
