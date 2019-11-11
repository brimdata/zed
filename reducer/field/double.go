package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/streamfn"
)

type Double struct {
	Field
	fn *streamfn.Float64
}

func NewDouble(name, field, op string) reducer.Interface {
	return &Double{
		Field: NewField(name, field),
		fn:    streamfn.NewFloat64(op),
	}
}

func (i *Double) Result() zeek.Value {
	return &zeek.Double{i.fn.State}
}

func (i *Double) Consume(r *zson.Record) {
	v := i.lookup(r)
	if v == nil {
		return
	}
	cv := zeek.CoerceToDouble(v)
	if cv == nil {
		i.TypeMismatch++
		return
	}
	i.fn.Update(cv.Native)
}
