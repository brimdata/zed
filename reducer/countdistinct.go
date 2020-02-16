package reducer

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

type CountDistinctProto struct {
	target   string
	resolver expr.FieldExprResolver
}

func (cdp *CountDistinctProto) Target() string {
	return cdp.target
}

func (cdp *CountDistinctProto) Instantiate(*zng.Record) Interface {
	return &CountDistinct{
		Resolver: cdp.resolver,
		sketch:   hyperloglog.New(),
	}
}

func NewCountDistinctProto(target string, resolver expr.FieldExprResolver) *CountDistinctProto {
	return &CountDistinctProto{target, resolver}
}

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	Reducer
	Resolver expr.FieldExprResolver
	sketch   *hyperloglog.Sketch
}

func (c *CountDistinct) Consume(r *zng.Record) {
	v := c.Resolver(r)
	c.sketch.Insert(v.Bytes)
}

func (c *CountDistinct) Result() zng.Value {
	return zng.NewUint64(c.sketch.Estimate())
}

// Sketch returns the native structure used to compute the distinct count
// approixmation. This method is exposed in case someone wants to merge the
// results with another CountDistinct reducer.
func (c *CountDistinct) Sketch() *hyperloglog.Sketch {
	return c.sketch
}
