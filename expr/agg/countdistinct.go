package agg

import (
	"errors"

	"github.com/axiomhq/hyperloglog"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	sketch *hyperloglog.Sketch
}

func NewCountDistinct() *CountDistinct {
	return &CountDistinct{
		sketch: hyperloglog.New(),
	}
}

func (c *CountDistinct) Consume(v zng.Value) error {
	c.sketch.Insert(v.Bytes)
	return nil
}

func (c *CountDistinct) Result(*resolver.Context) (zng.Value, error) {
	return zng.NewUint64(c.sketch.Estimate()), nil
}

func (*CountDistinct) ConsumeAsPartial(v zng.Value) error {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	return errors.New("partials not yet implemented in countdistinct")
}

func (*CountDistinct) ResultAsPartial(zctx *resolver.Context) (zng.Value, error) {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	return zng.Value{}, errors.New("partials not yet implemented in countdistinct")
}
