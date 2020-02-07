package field

import (
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
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
	if k, ok := zx.CoerceToInt(v); ok {
		i.fn.Update(k)
		return nil
	}
	return zng.ErrTypeMismatch
}
