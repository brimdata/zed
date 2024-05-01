package agg

import (
	"slices"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type CollectMap struct {
	arena   *zed.Arena
	entries map[string]mapEntry
	scratch []byte
}

func newCollectMap() *CollectMap {
	return &CollectMap{
		arena:   zed.NewArena(),
		entries: make(map[string]mapEntry),
	}
}

var _ Function = (*Collect)(nil)

type mapEntry struct {
	key zed.Value
	val zed.Value
}

func (c *CollectMap) Consume(val zed.Value) {
	if val.IsNull() {
		return
	}
	mtyp, ok := zed.TypeUnder(val.Type()).(*zed.TypeMap)
	if !ok {
		return
	}
	// Copy val.Bytes since we're going to keep slices of it.
	it := zcode.Iter(slices.Clone(val.Bytes()))
	for !it.Done() {
		keyTagAndBody := it.NextTagAndBody()
		key := valueUnder(c.arena, mtyp.KeyType, keyTagAndBody.Body())
		val := valueUnder(c.arena, mtyp.ValType, it.Next())
		c.scratch = zed.AppendTypeValue(c.scratch[:0], key.Type())
		c.scratch = append(c.scratch, keyTagAndBody...)
		// This will squash existing values which is what we want.
		c.entries[string(c.scratch)] = mapEntry{key, val}
	}
}

func (c *CollectMap) ConsumeAsPartial(val zed.Value) {
	c.Consume(val)
}

func (c *CollectMap) Result(zctx *zed.Context, arena *zed.Arena) zed.Value {
	if len(c.entries) == 0 {
		return zed.Null
	}
	var ktypes, vtypes []zed.Type
	for _, e := range c.entries {
		ktypes = append(ktypes, e.key.Type())
		vtypes = append(vtypes, e.val.Type())
	}
	// Keep track of number of unique types in collection. If there is only one
	// unique type we don't build a union for each value (though the base type could
	// be a union itself).
	ktyp, kuniq := unionOf(zctx, ktypes)
	vtyp, vuniq := unionOf(zctx, vtypes)
	var builder zcode.Builder
	for _, e := range c.entries {
		appendMapVal(&builder, ktyp, e.key, kuniq)
		appendMapVal(&builder, vtyp, e.val, vuniq)
	}
	typ := zctx.LookupTypeMap(ktyp, vtyp)
	b := zed.NormalizeMap(builder.Bytes())
	return arena.New(typ, b)
}

func (c *CollectMap) ResultAsPartial(zctx *zed.Context, arena *zed.Arena) zed.Value {
	return c.Result(zctx, arena)
}

func appendMapVal(b *zcode.Builder, typ zed.Type, val zed.Value, uniq int) {
	if uniq > 1 {
		u := zed.TypeUnder(typ).(*zed.TypeUnion)
		zed.BuildUnion(b, u.TagOf(val.Type()), val.Bytes())
	} else {
		b.Append(val.Bytes())
	}
}

func unionOf(zctx *zed.Context, types []zed.Type) (zed.Type, int) {
	types = zed.UniqueTypes(types)
	if len(types) == 1 {
		return types[0], 1
	}
	return zctx.LookupTypeUnion(types), len(types)
}

// valueUnder is like zed.(*Value).Under but it preserves non-union named types.
func valueUnder(arena *zed.Arena, typ zed.Type, b zcode.Bytes) zed.Value {
	val := arena.New(typ, b)
	if _, ok := zed.TypeUnder(typ).(*zed.TypeUnion); !ok {
		return val
	}
	return val.Under(arena)
}
