package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/streamfn"
)

type Count struct {
	fn *streamfn.Uint64
}

func NewCountStreamfn(op string) Streamfn {
	return &Count{
		fn: streamfn.NewUint64(op),
	}
}

func (i *Count) Result() zeek.Value {
	return &zeek.Count{i.fn.State}
}

func (i *Count) Consume(v zeek.Value) error {
	cv := zeek.CoerceToInt(v) //XXX need CoerceToCount?
	if cv == nil {
		return zson.ErrTypeMismatch
	}
	i.fn.Update(uint64(cv.Native))
	return nil
}
