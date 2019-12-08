package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
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

func (c *Count) Result() zeek.Value {
	return zeek.NewCount(c.fn.State)
}

func (c *Count) Consume(v zeek.Value) error {
	var i zeek.Int
	//XXX need CoerceToCount?
	if !zeek.CoerceToInt(v, &i) {
		return zq.ErrTypeMismatch
	}
	c.fn.Update(uint64(i))
	return nil
}
