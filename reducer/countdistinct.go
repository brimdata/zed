package reducer

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	Reducer
	Field  string
	sketch *hyperloglog.Sketch
}

func NewCountDistinct(name, field string) *CountDistinct {
	return &CountDistinct{
		Reducer: New(name),
		Field:   field,
		sketch:  hyperloglog.New(),
	}
}

func (c *CountDistinct) Consume(r *zson.Record) {
	i, ok := r.Descriptor.LUT[c.Field]
	if !ok {
		return
	}
	v := r.Slice(i)
	c.sketch.Insert(v)
}

func (c *CountDistinct) Result() zeek.Value {
	return &zeek.Count{Native: c.sketch.Estimate()}
}

// Sketch returns the native structure used to compute the distinct count
// approixmation. This method is exposed in case someone wants to merge the
// results with another CountDistinct reducer.
func (c *CountDistinct) Sketch() *hyperloglog.Sketch {
	return c.sketch
}
