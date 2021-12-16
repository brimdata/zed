package agg

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/brimdata/zed"
)

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	sketch *hyperloglog.Sketch
}

var _ Function = (*CountDistinct)(nil)

func NewCountDistinct() *CountDistinct {
	return &CountDistinct{
		sketch: hyperloglog.New(),
	}
}

func (c *CountDistinct) Consume(v zed.Value) {
	c.sketch.Insert(v.Bytes)
}

func (c *CountDistinct) Result(*zed.Context) zed.Value {
	return zed.NewUint64(c.sketch.Estimate())
}

func (*CountDistinct) ConsumeAsPartial(v zed.Value) {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	panic("countdistinct: partials not yet implemented")
}

func (*CountDistinct) ResultAsPartial(zctx *zed.Context) zed.Value {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	panic("countdistinct: partials not yet implemented")
}
