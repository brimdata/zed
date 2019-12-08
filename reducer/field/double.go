package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
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
	return zeek.NewDouble(i.fn.State)
}

func (i *Double) Consume(v zeek.Value) error {
	//XXX change this to use *zeek.Double
	var d zeek.Double
	if !zeek.CoerceToDouble(v, &d) {
		return zq.ErrTypeMismatch
	}
	i.fn.Update(float64(d))
	return nil
}
