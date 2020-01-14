package field

import (
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zx"
)

type Double struct {
	fn *streamfn.Float64
}

func NewDoubleStreamfn(op string) Streamfn {
	return &Double{
		fn: streamfn.NewFloat64(op),
	}
}

func (i *Double) Result() zng.Value {
	return zng.NewDouble(i.fn.State)
}

func (i *Double) Consume(v zng.Value) error {
	d, ok := zx.CoerceToDouble(v)
	if !ok {
		return zbuf.ErrTypeMismatch
	}
	i.fn.Update(float64(d))
	return nil
}
