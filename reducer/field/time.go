package field

import (
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zx"
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
	if ts, ok := zx.CoerceToTime(v); ok {
		t.fn.Update(ts)
		return nil
	}
	return zng.ErrTypeMismatch
}
