package function

import (
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
		entryTypes: make(map[zed.Type]zed.Type),
		keyType:    zctx.LookupTypeArray(zed.TypeString),
		zctx:       zctx,
	}
}

func (n *Flatten) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	typ := zed.TypeRecordOf(val.Type)
	if typ == nil {
		return &val
	}
	inner := n.innerTypeOf(val.Bytes, typ.Columns)
	n.Reset()
	n.encode(typ.Columns, inner, field.Path{}, val.Bytes)
	return ctx.NewValue(n.zctx.LookupTypeArray(inner), n.Bytes())
}

func (n *Flatten) innerTypeOf(b zcode.Bytes, cols []zed.Column) zed.Type {
	types := n.appendTypes(nil, b, cols)
	unique := zed.UniqueTypes(types)
	if len(unique) == 1 {
		return unique[0]
	}
	return n.zctx.LookupTypeUnion(unique)
}

func (n *Flatten) appendTypes(types []zed.Type, b zcode.Bytes, cols []zed.Column) []zed.Type {
	it := b.Iter()
	for _, col := range cols {
		val := it.Next()
		if typ := zed.TypeRecordOf(col.Type); typ != nil && val != nil {
			types = n.appendTypes(types, val, typ.Columns)
			continue
		}
		typ, ok := n.entryTypes[col.Type]
		if !ok {
			typ = n.zctx.MustLookupTypeRecord([]zed.Column{
				zed.NewColumn("key", n.keyType),
				zed.NewColumn("value", col.Type),
			})
			n.entryTypes[col.Type] = typ
		}
		types = append(types, typ)
	}
	return types
}

func (n *Flatten) encode(cols []zed.Column, inner zed.Type, base field.Path, b zcode.Bytes) {
	it := b.Iter()
	for _, col := range cols {
		val := it.Next()
		key := append(base, col.Name)
		if typ := zed.TypeRecordOf(col.Type); typ != nil && val != nil {
			n.encode(typ.Columns, inner, key, val)
			continue
		}
		typ := n.entryTypes[col.Type]
		union, _ := inner.(*zed.TypeUnion)
		if union != nil {
			n.BeginContainer()
			n.Append(zed.EncodeInt(int64(union.TagOf(typ))))
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
