package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/streamfn"
)

type Time struct {
	fn *streamfn.Time
}

func NewTimeStreamfn(op string) Streamfn {
	return &Time{
		fn:    streamfn.NewTime(op),
	}
}

func (t *Time) Result() zeek.Value {
	return &zeek.Time{t.fn.State}
}

func (t *Time) Consume(v zeek.Value) error {
	cv := zeek.CoerceToTime(v)
	if cv == nil {
		return zson.ErrTypeMismatch
	}
	t.fn.Update(cv.Native)
	return nil
}
