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
	entryTypes map[zed.Type]zed.Type
	zctx       *zed.Context

	// This exists only to reduce memory allocations.
	types []zed.Type
}

func NewFlatten(zctx *zed.Context) *Flatten {
	return &Flatten{
		entryTypes: make(map[zed.Type]zed.Type),
		keyType:    zctx.LookupTypeArray(zed.TypeString),
		zctx:       zctx,
	}
}

func (n *Flatten) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0]
	typ := zed.TypeRecordOf(val.Type())
	if typ == nil {
		return val
	}
	inner := n.innerTypeOf(val.Bytes(), typ.Fields)
	n.Reset()
	n.encode(typ.Fields, inner, field.Path{}, val.Bytes())
	return zed.NewValue(n.zctx.LookupTypeArray(inner), n.Bytes())
}

func (n *Flatten) innerTypeOf(b zcode.Bytes, fields []zed.Field) zed.Type {
	n.types = n.appendTypes(n.types[:0], b, fields)
	unique := zed.UniqueTypes(n.types)
	if len(unique) == 1 {
		return unique[0]
	}
	return n.zctx.LookupTypeUnion(unique)
}

func (n *Flatten) appendTypes(types []zed.Type, b zcode.Bytes, fields []zed.Field) []zed.Type {
	it := b.Iter()
	for _, f := range fields {
		val := it.Next()
		if typ := zed.TypeRecordOf(f.Type); typ != nil && val != nil {
			types = n.appendTypes(types, val, typ.Fields)
			continue
		}
		typ, ok := n.entryTypes[f.Type]
		if !ok {
			typ = n.zctx.MustLookupTypeRecord([]zed.Field{
				zed.NewField("key", n.keyType),
				zed.NewField("value", f.Type),
			})
			n.entryTypes[f.Type] = typ
		}
		types = append(types, typ)
	}
	return types
}

func (n *Flatten) encode(fields []zed.Field, inner zed.Type, base field.Path, b zcode.Bytes) {
	it := b.Iter()
	for _, f := range fields {
		val := it.Next()
		key := append(base, f.Name)
		if typ := zed.TypeRecordOf(f.Type); typ != nil && val != nil {
			n.encode(typ.Fields, inner, key, val)
			continue
		}
		typ := n.entryTypes[f.Type]
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
