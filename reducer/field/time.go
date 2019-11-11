package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/streamfn"
)

type Time struct {
	Field
	fn *streamfn.Time
}

func NewTime(name, field, op string) reducer.Interface {
	return &Time{
		Field: NewField(name, field),
		fn:    streamfn.NewTime(op),
	}
}

func (t *Time) Result() zeek.Value {
	return &zeek.Time{t.fn.State}
}

func (t *Time) Consume(r *zson.Record) {
	v := t.lookup(r)
	if v == nil {
		return
	}
	cv := zeek.CoerceToTime(v)
	if cv == nil {
		t.TypeMismatch++
		return
	}
	t.fn.Update(cv.Native)
}
