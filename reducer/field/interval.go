package field

import (
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
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
	var interval zng.Interval
	if !zng.CoerceToInterval(v, &interval) {
		return zbuf.ErrTypeMismatch
	}
	i.fn.Update(int64(interval))
	return nil
}
