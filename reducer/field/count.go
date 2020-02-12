package field

import (
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
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
	if v, ok := zx.CoerceToUint(v, zng.TypeUint64); ok {
		i, err := zng.DecodeUint(v.Bytes)
		if err != nil {
			return zng.ErrBadValue
		}
		c.fn.Update(i)
		return nil
	}
	return zng.ErrTypeMismatch
}
