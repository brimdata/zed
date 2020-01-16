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

func (d *Double) Result() zng.Value {
	return zng.NewDouble(d.fn.State)
}

func (d *Double) Consume(v zng.Value) error {
	dd, ok := zx.CoerceToDouble(v)
	if !ok {
		return zbuf.ErrTypeMismatch
	}
	d.fn.Update(dd)
	return nil
}
