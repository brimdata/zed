package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#unflatten
type Unflatten struct {
	zctx *zed.Context

	builder     zcode.Builder
	recordCache recordCache

	// These exist only to reduce memory allocations.
	path   field.Path
	types  []zed.Type
	values []zcode.Bytes
}

func NewUnflatten(zctx *zed.Context) *Unflatten {
	return &Unflatten{
		zctx: zctx,
	}
}

func (u *Unflatten) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	array, ok := zed.TypeUnder(val.Type).(*zed.TypeArray)
	if !ok {
		return &val
	}
	u.recordCache.reset()
	root := u.recordCache.new()
	u.types = u.types[:0]
	u.values = u.values[:0]
	for it := val.Bytes.Iter(); !it.Done(); {
		path, typ, vb := u.parseElem(array.Type, it.Next())
		if typ == nil {
			continue
		}
		root.addPath(&u.recordCache, path)
		u.types = append(u.types, typ)
		u.values = append(u.values, vb)
	}
	u.builder.Reset()
	types, values := u.types, u.values
	typ := root.build(u.zctx, &u.builder, func() (zed.Type, zcode.Bytes) {
		typ, value := types[0], values[0]
		types, values = types[1:], values[1:]
		return typ, value
	})
	return ctx.NewValue(typ, u.builder.Bytes())
}

func (u *Unflatten) parseElem(inner zed.Type, vb zcode.Bytes) (field.Path, zed.Type, zcode.Bytes) {
	if union, ok := zed.TypeUnder(inner).(*zed.TypeUnion); ok {
		inner, vb = union.SplitZNG(vb)
	}
	typ := zed.TypeRecordOf(inner)
	if typ == nil || len(typ.Columns) != 2 {
		return nil, nil, nil
	}
	nkey, ok := typ.ColumnOfField("key")
	if !ok {
		return nil, nil, nil
	}
	if a, ok := zed.TypeUnder(typ.Columns[nkey].Type).(*zed.TypeArray); !ok && a.Type != zed.TypeString {
		return nil, nil, nil
	}
	vtyp, ok := typ.TypeOfField("value")
	if !ok {
		return nil, nil, nil
	}
	it := vb.Iter()
	kbytes, vbytes := it.Next(), it.Next()
	if nkey == 1 {
		kbytes, vbytes = vbytes, kbytes
	}
	return u.decodeKey(kbytes), vtyp, vbytes
}

func (u *Unflatten) decodeKey(b zcode.Bytes) field.Path {
	u.path = u.path[:0]
	for it := b.Iter(); !it.Done(); {
		u.path = append(u.path, zed.DecodeString(it.Next()))
	}
	return u.path
}

type recordCache struct {
	index   int
	records []*record
}

func (c *recordCache) new() *record {
	if c.index == len(c.records) {
		c.records = append(c.records, new(record))
	}
	r := c.records[c.index]
	r.columns = r.columns[:0]
	r.records = r.records[:0]
	c.index++
	return r
}

func (c *recordCache) reset() {
	c.index = 0
}

type record struct {
	columns []zed.Column
	records []*record
}

func (r *record) addPath(c *recordCache, p []string) {
	if len(p) == 0 {
		return
	}
	if len(r.columns) == 0 || r.columns[len(r.columns)-1].Name != p[0] {
		r.appendColumn(p[0])
		var rec *record
		if len(p) > 1 {
			rec = c.new()
		}
		r.records = append(r.records, rec)
	}
	r.records[len(r.records)-1].addPath(c, p[1:])
}

func (r *record) appendColumn(name string) {
	if len(r.columns) == cap(r.columns) {
		r.columns = append(r.columns, zed.Column{})
	} else {
		r.columns = r.columns[:len(r.columns)+1]
	}
	r.columns[len(r.columns)-1].Name = name
}

func (r *record) build(zctx *zed.Context, b *zcode.Builder, next func() (zed.Type, zcode.Bytes)) zed.Type {
	for i, rec := range r.records {
		if rec == nil {
			typ, value := next()
			b.Append(value)
			r.columns[i].Type = typ
			continue
		}
		b.BeginContainer()
		r.columns[i].Type = rec.build(zctx, b, next)
		b.EndContainer()
	}
	return zctx.MustLookupTypeRecord(r.columns)
}
