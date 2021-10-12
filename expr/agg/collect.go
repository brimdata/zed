package agg

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Collect struct {
	values []zed.Value
	size   int
}

func (c *Collect) Consume(v zed.Value) error {
	if v.IsNil() {
		return nil
	}
	c.update(v)
	return nil
}

func (c *Collect) update(v zed.Value) {
	stash := make(zcode.Bytes, len(v.Bytes))
	copy(stash, v.Bytes)
	c.values = append(c.values, zed.Value{v.Type, stash})
	c.size += len(v.Bytes)
	for c.size > MaxValueSize {
		// XXX See issue #1813.  For now we silently discard entries
		// to maintain the size limit.
		//c.MemExceeded++
		c.size -= len(c.values[0].Bytes)
		c.values = c.values[1:]
	}
}

func (c *Collect) Result(zctx *zed.Context) (zed.Value, error) {
	if len(c.values) == 0 {
		// no values found
		return zed.Value{Type: zed.TypeNull}, nil
	}
	m := make(map[zed.Type]int)
	for _, zv := range c.values {
		m[zv.Type] = 0
	}
	if len(m) == 1 {
		return c.build(zctx)
	}
	return c.buildUnion(zctx, m)
}

func (c *Collect) build(zctx *zed.Context) (zed.Value, error) {
	typ := c.values[0].Type
	var b zcode.Builder
	container := zed.IsContainerType(typ)
	for _, v := range c.values {
		if container {
			b.AppendContainer(v.Bytes)
		} else {
			b.AppendPrimitive(v.Bytes)
		}
	}
	arrayType := zctx.LookupTypeArray(typ)
	return zed.Value{arrayType, b.Bytes()}, nil
}

func (c *Collect) buildUnion(zctx *zed.Context, selectors map[zed.Type]int) (zed.Value, error) {
	// XXX When all of the types are unions we should combine them into a
	// a merged union.  This will allow partials that compute different
	// unions to do the right thing.  See issue #3171.
	// XXX This map needs to be put in canonical order but we haven't
	// done canonical type order of unions yet.  See #2145.
	// Also, we should have a nice Zed library function
	// to create union values instead of doing the work here to find the
	// index of the type.
	types := make([]zed.Type, 0, len(selectors))
	for typ := range selectors {
		selectors[typ] = len(types)
		types = append(types, typ)
	}
	var b zcode.Builder
	for _, v := range c.values {
		selector := selectors[v.Type]
		b.BeginContainer()
		b.AppendPrimitive(zed.EncodeInt(int64(selector)))
		if zed.IsContainerType(v.Type) {
			b.AppendContainer(v.Bytes)
		} else {
			b.AppendPrimitive(v.Bytes)
		}
		b.EndContainer()
	}
	unionType := zctx.LookupTypeUnion(types)
	arrayType := zctx.LookupTypeArray(unionType)
	return zed.Value{arrayType, b.Bytes()}, nil
}

func (c *Collect) ConsumeAsPartial(zv zed.Value) error {
	//XXX These should not be passed in here. See issue #3175
	if len(zv.Bytes) == 0 {
		return nil
	}
	arrayType, ok := zv.Type.(*zed.TypeArray)
	if !ok {
		return errors.New("partial is not an array type in collect aggregator: " + zv.String())
	}
	typ := arrayType.Type
	for it := zv.Iter(); !it.Done(); {
		b, _, err := it.Next()
		if err != nil {
			return err
		}
		c.update(zed.Value{typ, b})
	}
	return nil
}

func (c *Collect) ResultAsPartial(zctx *zed.Context) (zed.Value, error) {
	return c.Result(zctx)
}
