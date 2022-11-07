package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type recordExpr struct {
	zctx    *zed.Context
	typ     *zed.TypeRecord
	builder *zcode.Builder
	columns []zed.Column
	exprs   []Evaluator
}

func NewRecordExpr(zctx *zed.Context, elems []RecordElem) (Evaluator, error) {
	for _, e := range elems {
		if e.Spread != nil {
			return newRecordSpreadExpr(zctx, elems)
		}
	}
	return newRecordExpr(zctx, elems), nil
}

func newRecordExpr(zctx *zed.Context, elems []RecordElem) *recordExpr {
	columns := make([]zed.Column, 0, len(elems))
	exprs := make([]Evaluator, 0, len(elems))
	for _, elem := range elems {
		columns = append(columns, zed.Column{Name: elem.Name})
		exprs = append(exprs, elem.Field)
	}
	var typ *zed.TypeRecord
	if len(exprs) == 0 {
		typ = zctx.MustLookupTypeRecord([]zed.Column{})
	}
	return &recordExpr{
		zctx:    zctx,
		typ:     typ,
		builder: zcode.NewBuilder(),
		columns: columns,
		exprs:   exprs,
	}
}

func (r *recordExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	var changed bool
	b := r.builder
	b.Reset()
	for k, e := range r.exprs {
		zv := e.Eval(ectx, this)
		if r.columns[k].Type != zv.Type {
			r.columns[k].Type = zv.Type
			changed = true
		}
		b.Append(zv.Bytes)
	}
	if changed {
		r.typ = r.zctx.MustLookupTypeRecord(r.columns)
	}
	bytes := b.Bytes()
	if bytes == nil {
		// Return empty record instead of null record.
		bytes = []byte{}
	}
	return ectx.NewValue(r.typ, bytes)
}

type RecordElem struct {
	Name   string
	Field  Evaluator
	Spread Evaluator
}

type recordSpreadExpr struct {
	zctx    *zed.Context
	elems   []RecordElem
	builder zcode.Builder
	columns []zed.Column
	bytes   []zcode.Bytes
	cache   *zed.TypeRecord
}

func newRecordSpreadExpr(zctx *zed.Context, elems []RecordElem) (*recordSpreadExpr, error) {
	return &recordSpreadExpr{
		zctx:  zctx,
		elems: elems,
	}, nil
}

type column struct {
	colno int
	value zed.Value
}

func (r *recordSpreadExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	object := make(map[string]column)
	for _, elem := range r.elems {
		if elem.Spread != nil {
			rec := elem.Spread.Eval(ectx, this)
			if rec.IsMissing() {
				continue
			}
			typ := zed.TypeRecordOf(rec.Type)
			if typ == nil {
				// Treat non-record spread values like missing.
				continue
			}
			it := rec.Iter()
			for _, col := range typ.Columns {
				c, ok := object[col.Name]
				if ok {
					c.value = zed.Value{Type: col.Type, Bytes: it.Next()}
				} else {
					c = column{
						colno: len(object),
						value: zed.Value{Type: col.Type, Bytes: it.Next()},
					}
				}
				object[col.Name] = c
			}
		} else {
			val := elem.Field.Eval(ectx, this)
			c, ok := object[elem.Name]
			if ok {
				c.value = *val
			} else {
				c = column{colno: len(object), value: *val}
			}
			object[elem.Name] = c
		}
	}
	if len(object) == 0 {
		b := r.builder
		b.Reset()
		b.Append(nil)
		return ectx.NewValue(r.zctx.MustLookupTypeRecord([]zed.Column{}), b.Bytes())
	}
	r.update(object)
	b := r.builder
	b.Reset()
	for _, bytes := range r.bytes {
		b.Append(bytes)
	}
	return ectx.NewValue(r.cache, b.Bytes())
}

// update maps the object into the receiver's vals slice while also
// seeing if we can reuse the cached record type.  If not we look up
// a new type, cache it, and save the columns for the cache check.
func (r *recordSpreadExpr) update(object map[string]column) {
	if len(r.columns) != len(object) {
		r.invalidate(object)
		return
	}
	for name, field := range object {
		col := zed.Column{Name: name, Type: field.value.Type}
		if r.columns[field.colno] != col {
			r.invalidate(object)
			return
		}
		r.bytes[field.colno] = field.value.Bytes
	}
}

func (r *recordSpreadExpr) invalidate(object map[string]column) {
	if n := len(object); cap(r.columns) < n {
		r.columns = make([]zed.Column, n)
		r.bytes = make([]zcode.Bytes, n)
	} else {
		r.columns = r.columns[:n]
		r.bytes = r.bytes[:n]
	}
	for name, field := range object {
		r.columns[field.colno] = zed.Column{Name: name, Type: field.value.Type}
		r.bytes[field.colno] = field.value.Bytes
	}
	r.cache = r.zctx.MustLookupTypeRecord(r.columns)
}

type VectorElem struct {
	Value  Evaluator
	Spread Evaluator
}

type ArrayExpr struct {
	elems []VectorElem
	zctx  *zed.Context

	builder    zcode.Builder
	collection collectionBuilder
}

func NewArrayExpr(zctx *zed.Context, elems []VectorElem) *ArrayExpr {
	return &ArrayExpr{
		elems: elems,
		zctx:  zctx,
	}
}

func (a *ArrayExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	a.builder.Reset()
	a.collection.reset()
	for _, e := range a.elems {
		if e.Value != nil {
			a.collection.append(e.Value.Eval(ectx, this))
			continue
		}
		val := e.Spread.Eval(ectx, this)
		inner := zed.InnerType(val.Type)
		if inner == nil {
			// Treat non-list spread values values like missing.
			continue
		}
		a.collection.appendSpread(inner, val.Bytes)
	}
	if len(a.collection.types) == 0 {
		return ectx.NewValue(a.zctx.LookupTypeArray(zed.TypeNull), []byte{})
	}
	it := a.collection.iter(a.zctx)
	for !it.done() {
		it.appendNext(&a.builder)
	}
	return ectx.NewValue(a.zctx.LookupTypeArray(it.typ), a.builder.Bytes())
}

type SetExpr struct {
	builder    zcode.Builder
	collection collectionBuilder
	elems      []VectorElem
	zctx       *zed.Context
}

func NewSetExpr(zctx *zed.Context, elems []VectorElem) *SetExpr {
	return &SetExpr{
		elems: elems,
		zctx:  zctx,
	}
}

func (a *SetExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	a.builder.Reset()
	a.collection.reset()
	for _, e := range a.elems {
		if e.Value != nil {
			a.collection.append(e.Value.Eval(ectx, this))
			continue
		}
		val := e.Spread.Eval(ectx, this)
		inner := zed.InnerType(val.Type)
		if inner == nil {
			// Treat non-list spread values values like missing.
			continue
		}
		a.collection.appendSpread(inner, val.Bytes)
	}
	if len(a.collection.types) == 0 {
		return ectx.NewValue(a.zctx.LookupTypeSet(zed.TypeNull), []byte{})
	}
	it := a.collection.iter(a.zctx)
	for !it.done() {
		it.appendNext(&a.builder)
	}
	return ectx.NewValue(a.zctx.LookupTypeSet(it.typ), zed.NormalizeSet(a.builder.Bytes()))
}

type Entry struct {
	Key Evaluator
	Val Evaluator
}

type MapExpr struct {
	builder zcode.Builder
	entries []Entry
	keys    collectionBuilder
	vals    collectionBuilder
	zctx    *zed.Context
}

func NewMapExpr(zctx *zed.Context, entries []Entry) *MapExpr {
	return &MapExpr{
		entries: entries,
		zctx:    zctx,
	}
}

func (m *MapExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	m.keys.reset()
	m.vals.reset()
	for _, e := range m.entries {
		m.keys.append(e.Key.Eval(ectx, this))
		m.vals.append(e.Val.Eval(ectx, this))
	}
	if len(m.keys.types) == 0 {
		typ := m.zctx.LookupTypeMap(zed.TypeNull, zed.TypeNull)
		return ectx.NewValue(typ, []byte{})
	}
	m.builder.Reset()
	kIter, vIter := m.keys.iter(m.zctx), m.vals.iter(m.zctx)
	for !kIter.done() {
		kIter.appendNext(&m.builder)
		vIter.appendNext(&m.builder)
	}
	bytes := m.builder.Bytes()
	typ := m.zctx.LookupTypeMap(kIter.typ, vIter.typ)
	return ectx.NewValue(typ, zed.NormalizeMap(bytes))
}

type collectionBuilder struct {
	types       []zed.Type
	uniqueTypes []zed.Type
	bytes       []zcode.Bytes
}

func (c *collectionBuilder) reset() {
	c.types = c.types[:0]
	c.uniqueTypes = c.uniqueTypes[:0]
	c.bytes = c.bytes[:0]
}

func (c *collectionBuilder) append(val *zed.Value) {
	c.types = append(c.types, val.Type)
	c.bytes = append(c.bytes, val.Bytes)
}

func (c *collectionBuilder) appendSpread(inner zed.Type, b zcode.Bytes) {
	for it := b.Iter(); !it.Done(); {
		c.types = append(c.types, inner)
		c.bytes = append(c.bytes, it.Next())
	}
}

func (c *collectionBuilder) iter(zctx *zed.Context) collectionIter {
	// uniqueTypes must be copied since zed.UniqueTypes operates on the type
	// array in place and thus we'll lose order.
	c.uniqueTypes = append(c.uniqueTypes[:0], c.types...)
	return collectionIter{
		typ:   unionOf(zctx, c.uniqueTypes),
		bytes: c.bytes,
		types: c.types,
		uniq:  len(c.uniqueTypes),
	}
}

type collectionIter struct {
	typ   zed.Type
	bytes []zcode.Bytes
	types []zed.Type
	uniq  int
}

func (c *collectionIter) appendNext(b *zcode.Builder) {
	if union, ok := c.typ.(*zed.TypeUnion); ok && c.uniq > 1 {
		zed.BuildUnion(b, union.TagOf(c.types[0]), c.bytes[0])
	} else {
		b.Append(c.bytes[0])
	}
	c.bytes = c.bytes[1:]
	c.types = c.types[1:]
}

func (c *collectionIter) done() bool {
	return len(c.types) == 0
}

func unionOf(zctx *zed.Context, types []zed.Type) zed.Type {
	unique := types[:0]
	for _, t := range zed.UniqueTypes(types) {
		if t != zed.TypeNull {
			unique = append(unique, t)
		}
	}
	if len(unique) == 0 {
		return zed.TypeNull
	}
	if len(unique) == 1 {
		return unique[0]
	}
	return zctx.LookupTypeUnion(unique)
}
