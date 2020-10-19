package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Count struct {
	Reducer
	arg   expr.Evaluator
	count uint64
}

func (c *Count) Consume(r *zng.Record) {
	if c.filter(r) {
		return
	}
	if c.arg != nil {
		if v, err := c.arg.Eval(r); err != nil || v.IsNil() {
			return
		}
	}
	c.count++
}

func (c *Count) Result() zng.Value {
	return zng.NewUint64(c.count)
}

func (c *Count) ConsumePart(p zng.Value) error {
	u, err := zng.DecodeUint(p.Bytes)
	if err != nil {
		return err
	}
	c.count += u
	return nil
}

func (c *Count) ResultPart(*resolver.Context) (zng.Value, error) {
	return c.Result(), nil
}
