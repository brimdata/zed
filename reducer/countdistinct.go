package reducer

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/mccanne/zq/zng"
)

type CountDistinctProto struct {
	target string
	field  string
}

func (cdp *CountDistinctProto) Target() string {
	return cdp.target
}

func (cdp *CountDistinctProto) Instantiate() Interface {
	return &CountDistinct{
		Field:  cdp.field,
		sketch: hyperloglog.New(),
	}
}

func NewCountDistinctProto(target, field string) *CountDistinctProto {
	return &CountDistinctProto{target, field}
}

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	Reducer
	Field  string
	sketch *hyperloglog.Sketch
}

func (c *CountDistinct) Consume(r *zng.Record) {
	colno, ok := r.Type.ColumnOfField(c.Field)
	if !ok {
		return
	}
	//XXX this isn't right
	v, _ := r.Slice(colno)
	c.sketch.Insert(v)
}

func (c *CountDistinct) Result() zng.Value {
	return zng.NewCount(c.sketch.Estimate())
}

// Sketch returns the native structure used to compute the distinct count
// approixmation. This method is exposed in case someone wants to merge the
// results with another CountDistinct reducer.
func (c *CountDistinct) Sketch() *hyperloglog.Sketch {
	return c.sketch
}
