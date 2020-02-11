package field

import (
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
)

type Double struct {
	fn *streamfn.Float64
}

func NewFloat64Streamfn(op string) Streamfn {
	return &Double{
		fn: streamfn.NewFloat64(op),
	}
}

func (d *Double) Result() zng.Value {
	return zng.NewFloat64(d.fn.State)
}

func (d *Double) Consume(v zng.Value) error {
	dd, ok := zx.CoerceToFloat64(v)
	if !ok {
		return zng.ErrTypeMismatch
	}
	d.fn.Update(dd)
	return nil
}
