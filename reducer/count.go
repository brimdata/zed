package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type CountProto struct {
	target              string
	resolver, tresolver expr.FieldExprResolver
}

func (cp *CountProto) Target() string {
	return cp.target
}

func (cp *CountProto) TargetResolver() expr.FieldExprResolver {
	return cp.tresolver
}

func (cp *CountProto) Instantiate() Interface {
	return &Count{Resolver: cp.resolver}
}

func NewCountProto(target string, tresolver, resolver expr.FieldExprResolver) *CountProto {
	return &CountProto{
		target:    target,
		resolver:  resolver,
		tresolver: tresolver,
	}
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
