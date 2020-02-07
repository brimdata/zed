package field

import (
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
)

type Interval struct {
	fn *streamfn.Int64
}

func NewIntervalStreamfn(op string) Streamfn {
	return &Interval{
		fn: streamfn.NewInt64(op),
	}
}

func (i *Interval) Result() zng.Value {
	return zng.NewInterval(i.fn.State)
}

func (i *Interval) Consume(v zng.Value) error {
	if interval, ok := zx.CoerceToInterval(v); ok {
		i.fn.Update(interval)
		return nil
	}
	return zng.ErrTypeMismatch
}
