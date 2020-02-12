package field

import (
	"github.com/brimsec/zq/streamfn"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
)

type Uint struct {
	fn *streamfn.Uint64
}

func NewUintStreamfn(op string) Streamfn {
	return &Uint{
		fn: streamfn.NewUint64(op),
	}
}

func (u *Uint) Result() zng.Value {
	return zng.NewUint64(u.fn.State)
}

func (u *Uint) Consume(v zng.Value) error {
	if v, ok := zx.CoerceToUint(v, zng.TypeUint64); ok {
		i, err := zng.DecodeUint(v.Bytes)
		if err != nil {
			return zng.ErrBadValue
		}
		u.fn.Update(i)
		return nil
	}
	return zng.ErrTypeMismatch
}
