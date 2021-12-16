package agg

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Collect struct {
	values []zed.Value
	size   int
}

var _ Function = (*Collect)(nil)

func (c *Collect) Consume(val *zed.Value) {
	if !val.IsNull() {
		c.update(val)
	}
}

func (c *Collect) update(val *zed.Value) {
	stash := make(zcode.Bytes, len(val.Bytes))
	copy(stash, val.Bytes)
	c.values = append(c.values, zed.Value{val.Type, stash})
	c.size += len(val.Bytes)
	for c.size > MaxValueSize {
		// XXX See issue #1813.  For now we silently discard entries
		// to maintain the size limit.
		//c.MemExceeded++
		c.size -= len(c.values[0].Bytes)
		c.values = c.values[1:]
	}
}

func (c *Collect) Result(zctx *zed.Context) *zed.Value {
	if len(c.values) == 0 {
		// no values found
		return zed.Null
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

func (c *Collect) build(zctx *zed.Context) *zed.Value {
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
	return zed.NewValue(arrayType, b.Bytes())
}

func (c *Collect) buildUnion(zctx *zed.Context, selectors map[zed.Type]int) *zed.Value {
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
	return zed.NewValue(arrayType, b.Bytes())
}

func (c *Collect) ConsumeAsPartial(val *zed.Value) {
	//XXX These should not be passed in here. See issue #3175
	if len(val.Bytes) == 0 {
		return
	}
	arrayType, ok := val.Type.(*zed.TypeArray)
	if !ok {
		panic(fmt.Errorf("collect partial: partial not an array type: %s", zson.MustFormatValue(*val)))
	}
	typ := arrayType.Type
	elem := zed.Value{Type: typ}
	for it := val.Iter(); !it.Done(); {
		b, _, err := it.Next()
		if err != nil {
			panic(fmt.Errorf("collect partial: array bytes are corrupt: %w", err))
		}
		elem.Bytes = b
		c.update(&elem)
	}
}

func (c *Collect) ResultAsPartial(zctx *zed.Context) *zed.Value {
	return c.Result(zctx)
}
