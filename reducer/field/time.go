package field

import (
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/streamfn"
)

type Time struct {
	fn *streamfn.Time
}

func NewTimeStreamfn(op string) Streamfn {
	return &Time{
		fn: streamfn.NewTime(op),
	}
}

func (t *Time) Result() zeek.Value {
	return zeek.NewTime(t.fn.State)
}

func (t *Time) Consume(v zeek.Value) error {
	var cv zeek.Time
	if !zeek.CoerceToTime(v, &cv) {
		return zson.ErrTypeMismatch
	}
	t.fn.Update(nano.Ts(cv))
	return nil
}
