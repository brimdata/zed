package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/streamfn"
)

type Double struct {
	fn *streamfn.Float64
}

func NewDoubleStreamfn(op string) Streamfn {
	return &Double{
		fn: streamfn.NewFloat64(op),
	}
}

func (i *Double) Result() zeek.Value {
	return &zeek.Double{i.fn.State}
}

func (i *Double) Consume(v zeek.Value) error {
	cv := zeek.CoerceToDouble(v)
	if cv == nil {
		return zson.ErrTypeMismatch
	}
	i.fn.Update(cv.Native)
	return nil
}
