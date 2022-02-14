package function

import (
	"fmt"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#flatten
type Flatten struct {
	zcode.Builder
	keyType    zed.Type
	mapper     *zed.Mapper
	entryTypes map[zed.Type]zed.Type
	zctx       *zed.Context
}

func NewFlatten(zctx *zed.Context) *Flatten {
	return &Flatten{
		entryTypes: make(map[int]zed.Type),
		keyType:    zctx.LookupTypeArray(zed.TypeString),
		mapper:     zed.NewMapper(zctx),
		zctx:       zctx,
	}
}

func (n *Flatten) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	typ := zed.TypeRecordOf(val.Type)
	if typ == nil {
		return &val
	}
	valType := n.mapper.Lookup(typ.ID())
	if valType == nil {
		types := n.collectTypes(typ.Columns)
		types = dedupeTypes(types)
		if len(types) == 1 {
			valType = types[0]
		} else {
			valType = n.zctx.LookupTypeUnion(types)
		}
		n.mapper.EnterType(typ.ID(), valType)
	}
	n.Reset()
	n.encode(typ.Columns, valType, field.Path{}, val.Bytes)
	return ctx.NewValue(n.zctx.LookupTypeArray(valType), n.Bytes())
}

func (n *Flatten) collectTypes(cols []zed.Column) []zed.Type {
	var types []zed.Type
	for _, col := range cols {
		if typ := zed.TypeRecordOf(col.Type); typ != nil {
			types = append(types, n.collectTypes(typ.Columns)...)
		} else {
			typ, ok := n.entryTypes[col.Type.ID()]
			if !ok {
				n.entryTypes[col.Type] = n.zctx.MustLookupTypeRecord([]zed.Column{
					zed.NewColumn("key", n.keyType),
					zed.NewColumn("value", col.Type),
				})
			}
			types = append(types, typ)
		}
	}
	return types
}

func dedupeTypes(types []zed.Type) []zed.Type {
	sort.SliceStable(types, func(i, j int) bool {
		return zed.CompareTypes(types[i], types[j]) < 0
	})
	out := make([]zed.Type, 0, len(types))
	var prev zed.Type
	for _, typ := range types {
		if typ != prev {
			out = append(out, typ)
			prev = typ
		}
	}
	return out
}

func (n *Flatten) encode(cols []zed.Column, inner zed.Type, base field.Path, b zcode.Bytes) {
	it := b.Iter()
	for _, col := range cols {
		val := it.Next()
		key := append(base, col.Name)
		if typ := zed.TypeRecordOf(col.Type); typ != nil {
			n.encode(typ.Columns, inner, key, val)
			continue
		}
		typ := n.entryTypes[col.Type.ID()]
		union, _ := inner.(*zed.TypeUnion)
		if union != nil {
			n.BeginContainer()
			n.Append(zed.EncodeInt(int64(union.Selector(typ))))
		}
		n.BeginContainer()
		n.encodeKey(key)
		n.Append(val)
		n.EndContainer()
		if union != nil {
			n.EndContainer()
		}
	}
}

func (n *Flatten) encodeKey(key field.Path) {
	n.BeginContainer()
	for _, name := range key {
		n.Append(zed.EncodeString(name))
	}
	n.EndContainer()
}
