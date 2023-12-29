package agg

import (
	"github.com/brimdata/zed"
)

type Count uint64

var _ Function = (*Count)(nil)

func (c *Count) Consume(*zed.Value) {
	*c++
}

func (c Count) Result(*zed.Context) *zed.Value {
	return zed.NewUint64(uint64(c))
}

func (c *Count) ConsumeAsPartial(partial *zed.Value) {
	if partial.Type() != zed.TypeUint64 {
		panic("count: partial not uint64")
	}
	*c += Count(partial.Uint())
}

func (c Count) ResultAsPartial(*zed.Context) *zed.Value {
	return c.Result(nil)
}
