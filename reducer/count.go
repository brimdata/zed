package reducer

import (
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/zng"
)

type CountProto struct {
	target   string
	resolver expr.FieldExprResolver
}

func (cp *CountProto) Target() string {
	return cp.target
}

func (cp *CountProto) Instantiate(*zng.Record) Interface {
	return &Count{Resolver: cp.resolver}
}

func NewCountProto(target string, resolver expr.FieldExprResolver) *CountProto {
	return &CountProto{target, resolver}
}

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
	return zng.NewCount(c.count)
}
