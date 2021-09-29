package agg

import (
	"errors"

	"github.com/axiomhq/hyperloglog"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
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

func (c *CountDistinct) Consume(v zed.Value) error {
	c.sketch.Insert(v.Bytes)
	return nil
}

func (c *CountDistinct) Result(*zson.Context) (zed.Value, error) {
	return zed.NewUint64(c.sketch.Estimate()), nil
}

func (*CountDistinct) ConsumeAsPartial(v zed.Value) error {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	return errors.New("partials not yet implemented in countdistinct")
}

func (*CountDistinct) ResultAsPartial(zctx *zson.Context) (zed.Value, error) {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	return zed.Value{}, errors.New("partials not yet implemented in countdistinct")
}
