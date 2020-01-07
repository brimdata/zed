package field

import (
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type Time struct {
	fn *streamfn.Time
}

func NewTimeStreamfn(op string) Streamfn {
	return &Time{
		fn: streamfn.NewTime(op),
	}
}

func (t *Time) Result() zng.Value {
	return zng.NewTime(t.fn.State)
}

func (t *Time) Consume(v zng.Value) error {
	var cv zng.Time
	if !zng.CoerceToTime(v, &cv) {
		return zbuf.ErrTypeMismatch
	}
	t.fn.Update(nano.Ts(cv))
	return nil
}
