package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/streamfn"
)

type Interval struct {
	Field
	fn *streamfn.Int64
}

func NewInterval(name, field, op string) reducer.Interface {
	return &Interval{
		Field: NewField(name, field),
		fn:    streamfn.NewInt64(op),
	}
}

func (i *Interval) Result() zeek.Value {
	return &zeek.Interval{i.fn.State}
}

func (i *Interval) Consume(r *zson.Record) {
	v := i.lookup(r)
	if v == nil {
		return
	}
	cv := zeek.CoerceToInterval(v)
	if cv == nil {
		i.TypeMismatch++
		return
	}
	i.fn.Update(cv.Native)
}
