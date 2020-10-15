package reducer

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
)

// CountDistinct uses hyperloglog to approximate the count of unique values for
// a field.
type CountDistinct struct {
	Reducer
	arg    expr.Evaluator
	sketch *hyperloglog.Sketch
}

func NewCountDistinct(arg, where expr.Evaluator) *CountDistinct {
	return &CountDistinct{
		Reducer: Reducer{where: where},
		arg:     arg,
		sketch:  hyperloglog.New(),
	}
}

func (c *CountDistinct) Consume(r *zng.Record) {
	if c.filter(r) {
		return
	}
	v, err := c.arg.Eval(r)
	if err == nil {
		c.sketch.Insert(v.Bytes)
	}
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
