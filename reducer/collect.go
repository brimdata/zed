package reducer

import (
	"errors"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Collect struct {
	Reducer
	zctx *resolver.Context
	arg  expr.Evaluator
	typ  zng.Type
	val  []zcode.Bytes
	size int
}

func (c *Collect) Consume(r *zng.Record) {
	if c.filter(r) {
		return
	}
	v, err := c.arg.Eval(r)
	if err != nil || v.IsNil() {
		return
	}
	if c.typ == nil {
		c.typ = v.Type
	} else if c.typ != v.Type {
		c.TypeMismatch++
		return
	}
	c.update(v.Bytes)
}

func (c *Collect) update(b zcode.Bytes) {
	c.val = append(c.val, b)
	c.size += len(b)
	for c.size > MaxValueSize {
		c.size -= len(c.val[0])
		c.val = c.val[1:]
	}
}

func (c *Collect) Result() zng.Value {
	if c.typ == nil {
		// must be empty array
		typ := c.zctx.LookupTypeArray(zng.TypeNull)
		return zng.Value{typ, nil}
	}
	var b zcode.Builder
	container := zng.IsContainerType(c.typ)
	for _, item := range c.val {
		if container {
			b.AppendContainer(item)
		} else {
			b.AppendPrimitive(item)
		}
	}
	typ := c.zctx.LookupTypeArray(c.typ)
	return zng.Value{typ, b.Bytes()}
}

func (c *Collect) ConsumePart(zv zng.Value) error {
	if c.typ == nil {
		typ, ok := zv.Type.(*zng.TypeArray)
		if !ok {
			return errors.New("partial not an array type")
		}
		c.typ = typ.Type
	}
	for it := zv.Iter(); !it.Done(); {
		elem, _, err := it.Next()
		if err != nil {
			return err
		}
		c.update(elem)
	}
	return nil
}

func (c *Collect) ResultPart(*resolver.Context) (zng.Value, error) {
	return c.Result(), nil
}
