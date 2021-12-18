package agg

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/brimdata/zed"
)

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	sketch *hyperloglog.Sketch
	stash  zed.Value
}

var _ Function = (*CountDistinct)(nil)

func NewCountDistinct() *CountDistinct {
	return &CountDistinct{
		sketch: hyperloglog.New(),
		stash:  zed.Value{Type: zed.TypeUint64},
	}
}

func (c *CountDistinct) Consume(val *zed.Value) {
	c.sketch.Insert(val.Bytes)
}

func (c *CountDistinct) Result(*zed.Context) *zed.Value {
	c.stash.Bytes = zed.EncodeUint(c.sketch.Estimate())
	return &c.stash
}

func (*CountDistinct) ConsumeAsPartial(*zed.Value) {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	panic("countdistinct: partials not yet implemented")
}

func (*CountDistinct) ResultAsPartial(zctx *zed.Context) *zed.Value {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	panic("countdistinct: partials not yet implemented")
}
