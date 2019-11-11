package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/streamfn"
)

type Int struct {
	Field
	fn *streamfn.Int64
}

func NewInt(name, field, op string) reducer.Interface {
	return &Int{
		Field: NewField(name, field),
		fn:    streamfn.NewInt64(op),
	}
}

func (i *Int) Result() zeek.Value {
	return &zeek.Int{i.fn.State}
}

func (i *Int) Consume(r *zson.Record) {
	v := i.lookup(r)
	if v == nil {
		return
	}
	cv := zeek.CoerceToInt(v)
	if cv == nil {
		i.TypeMismatch++
		return
	}
	i.fn.Update(cv.Native)
}
