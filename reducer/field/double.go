package field

import (
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
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
	//XXX change this to use *zng.Double
	var zd zng.Double
	if !zng.CoerceToDouble(v, &zd) {
		return zbuf.ErrTypeMismatch
	}
	d.fn.Update(float64(zd))
	return nil
}
