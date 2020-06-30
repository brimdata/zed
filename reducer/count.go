package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Count struct {
	Reducer
	Resolver expr.FieldExprResolver
	count    uint64
}

func (c *Count) Consume(r *zng.Record) {
	if c.Resolver != nil {
		if v := c.Resolver(r); v.IsNil() {
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
