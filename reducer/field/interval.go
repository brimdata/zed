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
	return &zeek.Interval{i.fn.State}
}

func (i *Interval) Consume(v zeek.Value) error {
	cv := zeek.CoerceToInterval(v)
	if cv == nil {
		return zson.ErrTypeMismatch
	}
	i.fn.Update(cv.Native)
	return nil
}
