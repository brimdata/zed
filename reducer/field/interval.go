package field

import (
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
)

type Duration struct {
	fn *streamfn.Int64
}

func NewDurationStreamfn(op string) Streamfn {
	return &Duration{
		fn: streamfn.NewInt64(op),
	}
}

func (d *Duration) Result() zng.Value {
	return zng.NewDuration(d.fn.State)
}

func (d *Duration) Consume(v zng.Value) error {
	if interval, ok := zx.CoerceToDuration(v); ok {
		d.fn.Update(interval)
		return nil
	}
	return zng.ErrTypeMismatch
}
