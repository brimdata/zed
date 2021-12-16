package agg

import (
	"github.com/brimdata/zed"
)

type Count uint64

var _ Function = (*Count)(nil)

func (c *Count) Consume(val *zed.Value) {
	if !val.IsNil() {
		*c++
	}
}

func (c Count) Result(*zed.Context) *zed.Value {
	//XXX should reuse zed.Value
	return zed.NewValue(zed.NewUint64(uint64(c)))
}

func (c *Count) ConsumeAsPartial(partial *zed.Value) {
	//XXX check type and panic
	u, err := zed.DecodeUint(partial.Bytes)
	if err == nil {
		*c += Count(u)
	}
}

func (c Count) ResultAsPartial(*zed.Context) *zed.Value {
	return c.Result(nil)
}
