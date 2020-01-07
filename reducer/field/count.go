package field

import (
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type Count struct {
	fn *streamfn.Uint64
}

func NewCountStreamfn(op string) Streamfn {
	return &Count{
		fn: streamfn.NewUint64(op),
	}
}

func (c *Count) Result() zng.Value {
	return zng.NewCount(c.fn.State)
}

func (c *Count) Consume(v zng.Value) error {
	var i zng.Int
	//XXX need CoerceToCount?
	if !zng.CoerceToInt(v, &i) {
		return zbuf.ErrTypeMismatch
	}
	c.fn.Update(uint64(i))
	return nil
}
