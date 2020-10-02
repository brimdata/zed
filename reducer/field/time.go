package field

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
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
	if ts, ok := expr.CoerceToTime(v); ok {
		t.fn.Update(ts)
		return nil
	}
	return zng.ErrTypeMismatch
}
