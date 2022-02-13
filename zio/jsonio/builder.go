package jsonio

import (
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type builder struct {
	zctx *zed.Context

	containers []int  // Stack of open containers (as indexes into items).
	items      []item // Stack of items.

	// These exist only to reduce memory allocations.
	bytes   []byte
	columns []zed.Column
	indices []int
	types   []zed.Type
}

type item struct {
	fieldName string
	typ       zed.Type
	zb        zcode.Builder
}

func (b *builder) reset() {
	b.containers = b.containers[:0]
	b.items = b.items[:0]
}

func (b *builder) pushPrimitiveItem(fieldName string, typ zed.Type, bytes zcode.Bytes) {
	i := b.pushItem(fieldName)
	i.typ = typ
	i.zb.Append(bytes)
}

func (b *builder) pushItem(fieldName string) *item {
	n := len(b.items)
	if n == cap(b.items) {
		b.items = append(b.items, item{})
	}
	b.items = b.items[:n+1]
	i := &b.items[n]
	i.fieldName = fieldName
	i.zb.Truncate()
	return i
}

func (b *builder) beginContainer(fieldName string) {
	// This item represents the container.  endArray or endRecord will set
	// its type and bytes.
	b.pushItem(fieldName)
	b.containers = append(b.containers, len(b.items))
}

func (b *builder) endContainer() (container *item, items []item) {
	start := b.containers[len(b.containers)-1]
	b.containers = b.containers[:len(b.containers)-1]
	items = b.items[start:]
	b.items = b.items[:start]
	return &b.items[start-1], items
}

func (b *builder) endArray() {
	container, items := b.endContainer()

	b.types = b.types[:0]
	for i := range items {
		if items[i].typ != zed.TypeNull {
			b.types = append(b.types, items[i].typ)
		}
	}
	sort.Slice(b.types, func(i, j int) bool { return b.types[i].ID() < b.types[j].ID() })
	dedupedTypes := b.types[:0]
	var prev zed.Type
	for _, t := range b.types {
		// JSON doesn't use named types, so even though b.types was
		// sorted by zed.Type.ID above, we can compare elements directly
		// without calling zed.TypeUnder.
		if t != prev {
			dedupedTypes = append(dedupedTypes, t)
			prev = t
		}
	}
	b.types = dedupedTypes

	container.zb.BeginContainer()
	switch len(b.types) {
	case 0:
		container.typ = b.zctx.LookupTypeArray(zed.TypeNull)
		for range items {
			container.zb.Append(nil)
		}
	case 1:
		container.typ = b.zctx.LookupTypeArray(b.types[0])
		for i := range items {
			container.zb.Append(trimTag(items[i].zb.Bytes()))
		}
	default:
		union := b.zctx.LookupTypeUnion(b.types)
		if &union.Types[0] == &b.types[0] {
			// union now owns b.types, so don't reuse it.
			b.types = nil
		}
		container.typ = b.zctx.LookupTypeArray(union)
		for i := range items {
			if bytes := trimTag(items[i].zb.Bytes()); bytes == nil {
				container.zb.Append(nil)
			} else {
				selector := union.Selector(items[i].typ)
				zed.BuildUnion(&container.zb, selector, bytes)
			}
		}
	}
	container.zb.EndContainer()
}

func (b *builder) endRecord() {
	container, items := b.endContainer()

	b.indices = b.indices[:0]
	for i := range items {
		b.indices = append(b.indices, i)
	}
	sort.SliceStable(b.indices, func(i, j int) bool {
		return items[b.indices[i]].fieldName < items[b.indices[j]].fieldName
	})
	dedupedIndices := b.indices[:0]
	prevIndex := -1
	for _, index := range b.indices {
		if prevIndex >= 0 && items[index].fieldName == items[prevIndex].fieldName {
			// Last occurence of a repeated field wins.
			dedupedIndices[len(dedupedIndices)-1] = index
		} else {
			dedupedIndices = append(dedupedIndices, index)
			prevIndex = index
		}
	}

	b.columns = b.columns[:0]
	container.zb.BeginContainer()
	for _, index := range dedupedIndices {
		item := &items[index]
		b.columns = append(b.columns, zed.Column{Name: item.fieldName, Type: item.typ})
		container.zb.Append(trimTag(item.zb.Bytes()))
	}
	container.zb.EndContainer()
	container.typ = b.zctx.MustLookupTypeRecord(b.columns)
}

func (b *builder) value() *zed.Value {
	if len(b.containers) > 0 {
		panic("open container")
	}
	if len(b.items) > 1 {
		panic("multiple items")
	}
	bytes := trimTag(b.items[0].zb.Bytes())
	// Reset gives us ownership of bytes.
	b.items[0].zb.Reset()
	return zed.NewValue(b.items[0].typ, bytes)
}

func trimTag(b zcode.Bytes) zcode.Bytes {
	b, err := b.Body()
	if err != nil {
		panic(err)
	}
	return b
}
