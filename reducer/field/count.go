package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/streamfn"
)

type Count struct {
	Field
	fn *streamfn.Uint64
}

func NewCount(name, field, op string) reducer.Interface {
	return &Count{
		Field: NewField(name, field),
		fn:    streamfn.NewUint64(op),
	}
}

func (i *Count) Result() zeek.Value {
	return &zeek.Count{i.fn.State}
}

func (i *Count) Consume(r *zson.Record) {
	v := i.lookup(r)
	if v == nil {
		return
	}
	cv := zeek.CoerceToInt(v) //XXX need CoerceToCount?
	if cv == nil {
		i.TypeMismatch++
		return
	}
	i.fn.Update(uint64(cv.Native))
}
