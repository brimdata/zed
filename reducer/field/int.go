package field

import (
	"github.com/mccanne/zq/streamfn"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type Int struct {
	fn *streamfn.Int64
}

func NewIntStreamfn(op string) Streamfn {
	return &Int{
		fn: streamfn.NewInt64(op),
	}
}

func (i *Int) Result() zng.Value {
	return zng.NewInt(i.fn.State)
}

func (i *Int) Consume(v zng.Value) error {
	var k zng.Int
	if !zng.CoerceToInt(v, &k) {
		return zbuf.ErrTypeMismatch
	}
	i.fn.Update(int64(k))
	return nil
}
