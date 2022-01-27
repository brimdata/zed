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
			it := rec.Bytes.Iter()
			for _, col := range typ.Columns {
				c, ok := object[col.Name]
				if ok {
					c.value = zed.Value{col.Type, it.Next()}
				} else {
					c = column{
						colno: len(object),
						value: zed.Value{col.Type, it.Next()},
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
		col := zed.Column{name, field.value.Type}
		if r.columns[field.colno] != col {
			r.invalidate(object)
			return
		}
		r.bytes[field.colno] = field.value.Bytes
	}
}

func (r *recordSpreadExpr) invalidate(object map[string]column) {
	n := len(object)
	if cap(r.columns) < n {
		r.columns = make([]zed.Column, n)
		r.bytes = make([]zcode.Bytes, n)
	} else {
		r.columns = r.columns[:n]
		r.bytes = r.bytes[:n]
	}
	for name, field := range object {
		r.columns[field.colno] = zed.Column{name, field.value.Type}
		r.bytes[field.colno] = field.value.Bytes
	}
	r.cache = r.zctx.MustLookupTypeRecord(r.columns)
}

type ArrayExpr struct {
	zctx    *zed.Context
	typ     *zed.TypeArray
	builder *zcode.Builder
	exprs   []Evaluator
}

func NewArrayExpr(zctx *zed.Context, exprs []Evaluator) *ArrayExpr {
	return &ArrayExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeArray(zed.TypeNull),
		builder: zcode.NewBuilder(),
		exprs:   exprs,
	}
}

func (a *ArrayExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	inner := a.typ.Type
	b := a.builder
	b.Reset()
	var first zed.Type
	for _, e := range a.exprs {
		zv := e.Eval(ectx, this)
		typ := zv.Type
		if first == nil {
			first = typ
		}
		if typ != inner && typ != zed.TypeNull {
			if typ == first || first == zed.TypeNull {
				a.typ = a.zctx.LookupTypeArray(zv.Type)
				inner = a.typ.Type
			} else {
				//XXX issue #3363
				return ectx.CopyValue(*a.zctx.NewErrorf("mixed-type array expressions not yet supported"))
			}
		}
		b.Append(zv.Bytes)
	}
	bytes := b.Bytes()
	if bytes == nil {
		// Return empty array instead of null array.
		bytes = []byte{}
	}
	return ectx.NewValue(a.typ, bytes)
}

type SetExpr struct {
	zctx    *zed.Context
	typ     *zed.TypeSet
	builder *zcode.Builder
	exprs   []Evaluator
}

func NewSetExpr(zctx *zed.Context, exprs []Evaluator) *SetExpr {
	return &SetExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeSet(zed.TypeNull),
		builder: zcode.NewBuilder(),
		exprs:   exprs,
	}
}

func (s *SetExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	var inner zed.Type
	b := s.builder
	b.Reset()
	var first zed.Type
	for _, e := range s.exprs {
		val := e.Eval(ectx, this)
		typ := val.Type
		if first == nil {
			first = typ
		}
		if typ != inner && typ != zed.TypeNull {
			if typ == first || first == zed.TypeNull {
				s.typ = s.zctx.LookupTypeSet(val.Type)
				inner = s.typ.Type
			} else {
				//XXX issue #3363
				return ectx.CopyValue(*s.zctx.NewErrorf("mixed-type set expressions not yet supported"))
			}
		}
		b.Append(val.Bytes)
	}
	bytes := b.Bytes()
	if bytes == nil {
		// Return empty set instead of null set.
		bytes = []byte{}
	}
	return ectx.NewValue(s.typ, zed.NormalizeSet(bytes))
}

type Entry struct {
	Key Evaluator
	Val Evaluator
}

type MapExpr struct {
	zctx    *zed.Context
	typ     *zed.TypeMap
	builder *zcode.Builder
	entries []Entry
}

func NewMapExpr(zctx *zed.Context, entries []Entry) *MapExpr {
	return &MapExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeMap(zed.TypeNull, zed.TypeNull),
		builder: zcode.NewBuilder(),
		entries: entries,
	}
}

func (m *MapExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	var keyType, valType zed.Type
	b := m.builder
	b.Reset()
	for _, e := range m.entries {
		key := e.Key.Eval(ectx, this)
		val := e.Val.Eval(ectx, this)
		if keyType == nil {
			if m.typ == nil || m.typ.KeyType != key.Type || m.typ.ValType != val.Type {
				keyType = key.Type
				valType = val.Type
				m.typ = m.zctx.LookupTypeMap(keyType, valType)
			} else {
				keyType = m.typ.KeyType
				valType = m.typ.ValType
			}
		} else if keyType != m.typ.KeyType || valType != m.typ.ValType {
			//XXX issue #3363
			return ectx.CopyValue(*m.zctx.NewErrorf("mixed-type map expressions not yet supported"))
		}
		b.Append(key.Bytes)
		b.Append(val.Bytes)
	}
	bytes := b.Bytes()
	if bytes == nil {
		// Return empty map instead of null map.
		bytes = []byte{}
	}
	return ectx.CopyValue(zed.Value{m.typ, zed.NormalizeMap(bytes)})
}
