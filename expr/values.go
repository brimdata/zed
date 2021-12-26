package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type RecordExpr struct {
	zctx    *zed.Context
	typ     *zed.TypeRecord
	builder *zcode.Builder
	columns []zed.Column
	exprs   []Evaluator
}

func NewRecordExpr(zctx *zed.Context, names []string, exprs []Evaluator) *RecordExpr {
	columns := make([]zed.Column, 0, len(names))
	for _, name := range names {
		columns = append(columns, zed.Column{Name: name})
	}
	var typ *zed.TypeRecord
	if len(exprs) == 0 {
		typ = zctx.MustLookupTypeRecord([]zed.Column{})
	}
	return &RecordExpr{
		zctx:    zctx,
		typ:     typ,
		builder: zcode.NewBuilder(),
		columns: columns,
		exprs:   exprs,
	}
}

func (r *RecordExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	var changed bool
	b := r.builder
	b.Reset()
	for k, e := range r.exprs {
		zv := e.Eval(ectx, this)
		if r.columns[k].Type != zv.Type {
			r.columns[k].Type = zv.Type
			changed = true
		}
		if zed.IsContainerType(zv.Type) {
			b.AppendContainer(zv.Bytes)
		} else {
			b.AppendPrimitive(zv.Bytes)
		}
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
	container := zed.IsContainerType(inner)
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
				container = zed.IsContainerType(inner)
			} else {
				//XXX issue #3363
				return ectx.CopyValue(*zed.NewErrorf("mixed-type array expressions not yet supported"))
			}
		}
		if container {
			b.AppendContainer(zv.Bytes)
		} else {
			b.AppendPrimitive(zv.Bytes)
		}
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
	var container bool
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
				container = zed.IsContainerType(inner)
			} else {
				//XXX issue #3363
				return ectx.CopyValue(*zed.NewErrorf("mixed-type set expressions not yet supported"))
			}
		}
		if container {
			b.AppendContainer(val.Bytes)
		} else {
			b.AppendPrimitive(val.Bytes)
		}
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
	var containerKey, containerVal bool
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
			containerKey = zed.IsContainerType(keyType)
			containerVal = zed.IsContainerType(valType)
		} else if keyType != m.typ.KeyType || valType != m.typ.ValType {
			//XXX issue #3363
			return ectx.CopyValue(*zed.NewErrorf("mixed-type map expressions not yet supported"))
		}
		if containerKey {
			b.AppendContainer(key.Bytes)
		} else {
			b.AppendPrimitive(key.Bytes)
		}
		if containerVal {
			b.AppendContainer(val.Bytes)
		} else {
			b.AppendPrimitive(val.Bytes)
		}
	}
	bytes := b.Bytes()
	if bytes == nil {
		// Return empty map instead of null map.
		bytes = []byte{}
	}
	return ectx.CopyValue(zed.Value{m.typ, zed.NormalizeMap(bytes)})
}
