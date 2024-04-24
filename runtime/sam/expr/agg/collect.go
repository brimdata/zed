package agg

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Collect struct {
	arena  *zed.Arena
	values []zed.Value
	size   int
}

var _ Function = (*Collect)(nil)

func newCollect() *Collect {
	return &Collect{arena: zed.NewArena()}
}

func (c *Collect) Consume(val zed.Value) {
	if !val.IsNull() {
		c.update(val)
	}
}

func (c *Collect) update(val zed.Value) {
	c.values = append(c.values, val.Under(c.arena).Copy(c.arena))
	c.size += len(val.Bytes())
	for c.size > MaxValueSize {
		// XXX See issue #1813.  For now we silently discard entries
		// to maintain the size limit.
		//c.MemExceeded++
		c.size -= len(c.values[0].Bytes())
		c.values = c.values[1:]
	}
}

func (c *Collect) Result(zctx *zed.Context, arena *zed.Arena) zed.Value {
	if len(c.values) == 0 {
		// no values found
		return zed.Null
	}
	var b zcode.Builder
	inner := innerType(zctx, c.values)
	if union, ok := inner.(*zed.TypeUnion); ok {
		for _, val := range c.values {
			zed.BuildUnion(&b, union.TagOf(val.Type()), val.Bytes())
		}
	} else {
		for _, val := range c.values {
			b.Append(val.Bytes())
		}
	}
	return arena.New(zctx.LookupTypeArray(inner), b.Bytes())
}

func innerType(zctx *zed.Context, vals []zed.Value) zed.Type {
	var types []zed.Type
	for _, val := range vals {
		types = append(types, val.Type())
	}
	types = zed.UniqueTypes(types)
	if len(types) == 1 {
		return types[0]
	}
	return zctx.LookupTypeUnion(types)
}

func (c *Collect) ConsumeAsPartial(_ *zed.Arena, val zed.Value) {
	//XXX These should not be passed in here. See issue #3175
	if len(val.Bytes()) == 0 {
		return
	}
	arrayType, ok := val.Type().(*zed.TypeArray)
	if !ok {
		panic(fmt.Errorf("collect partial: partial not an array type: %s", zson.FormatValue(val)))
	}
	typ := arrayType.Type
	for it := val.Iter(); !it.Done(); {
		c.update(c.arena.New(typ, it.Next()))
	}
}

func (c *Collect) ResultAsPartial(zctx *zed.Context, arena *zed.Arena) zed.Value {
	return c.Result(zctx, arena)
}
