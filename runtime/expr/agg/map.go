package agg

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/exp/slices"
)

type Map struct {
	entries map[string]mapEntry
	values  []zed.Value
	size    int
	scratch []byte
}

func newMap() *Map {
	return &Map{entries: make(map[string]mapEntry)}
}

var _ Function = (*Collect)(nil)

type mapEntry struct {
	key *zed.Value
	val *zed.Value
}

func (m *Map) Consume(val *zed.Value) {
	if val.IsNull() {
		return
	}
	mtyp, ok := zed.TypeUnder(val.Type).(*zed.TypeMap)
	if !ok {
		return
	}
	// Copy val.Bytes since we're going to keep slices of it.
	it := zcode.Iter(slices.Clone(val.Bytes))
	for !it.Done() {
		keyTagAndBody := it.NextTagAndBody()
		key := valueUnder(mtyp.KeyType, keyTagAndBody.Body())
		val := valueUnder(mtyp.ValType, it.Next())
		m.scratch = zed.AppendTypeValue(m.scratch[:0], key.Type)
		m.scratch = append(m.scratch, keyTagAndBody...)
		// This will squash existing values which is what we want.
		m.entries[string(m.scratch)] = mapEntry{key, val}
	}
}

func (m *Map) ConsumeAsPartial(val *zed.Value) {
	m.Consume(val)
}

func (m *Map) Result(zctx *zed.Context) *zed.Value {
	if len(m.entries) == 0 {
		return zed.Null
	}
	var ktypes, vtypes []zed.Type
	for _, e := range m.entries {
		ktypes = append(ktypes, e.key.Type)
		vtypes = append(vtypes, e.val.Type)
	}
	// Keep track of number of unique types in collection. If there is only one
	// unique type we don't build a union for each value (though the base type could
	// be a union itself).
	ktyp, kuniq := unionOf(zctx, ktypes)
	vtyp, vuniq := unionOf(zctx, vtypes)
	var builder zcode.Builder
	for _, e := range m.entries {
		appendMapVal(&builder, ktyp, e.key, kuniq)
		appendMapVal(&builder, vtyp, e.val, vuniq)
	}
	typ := zctx.LookupTypeMap(ktyp, vtyp)
	b := zed.NormalizeMap(builder.Bytes())
	return zed.NewValue(typ, b)
}

func (m *Map) ResultAsPartial(zctx *zed.Context) *zed.Value {
	return m.Result(zctx)
}

func appendMapVal(b *zcode.Builder, typ zed.Type, val *zed.Value, uniq int) {
	if uniq > 1 {
		u := zed.TypeUnder(typ).(*zed.TypeUnion)
		zed.BuildUnion(b, u.TagOf(val.Type), val.Bytes)
	} else {
		b.Append(val.Bytes)
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
func valueUnder(typ zed.Type, b zcode.Bytes) *zed.Value {
	val := zed.NewValue(typ, b)
	if _, ok := zed.TypeUnder(typ).(*zed.TypeUnion); !ok {
		return val
	}
	return val.Under()
}
