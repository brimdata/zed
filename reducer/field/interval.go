package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/streamfn"
)

type Interval struct {
	fn *streamfn.Int64
}

func NewIntervalStreamfn(op string) Streamfn {
	return &Interval{
		fn: streamfn.NewInt64(op),
	}
}

func (i *Interval) Result() zeek.Value {
	return zeek.NewInterval(i.fn.State)
}

func (i *Interval) Consume(v zeek.Value) error {
	var interval zeek.Interval
	if !zeek.CoerceToInterval(v, &interval) {
		return zson.ErrTypeMismatch
	}
	i.fn.Update(int64(interval))
	return nil
}
