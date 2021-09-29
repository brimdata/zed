package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type RecordExpr struct {
	zctx    *zson.Context
	typ     *zed.TypeRecord
	builder *zcode.Builder
	columns []zed.Column
	exprs   []Evaluator
}

func NewRecordExpr(zctx *zson.Context, names []string, exprs []Evaluator) *RecordExpr {
	columns := make([]zed.Column, 0, len(names))
	for _, name := range names {
		columns = append(columns, zed.Column{Name: name})
	}
	return &RecordExpr{
		zctx:    zctx,
		builder: zcode.NewBuilder(),
		columns: columns,
		exprs:   exprs,
	}
}

func (r *RecordExpr) Eval(rec *zed.Record) (zed.Value, error) {
	var changed bool
	b := r.builder
	b.Reset()
	for k, e := range r.exprs {
		zv, err := e.Eval(rec)
		if err != nil {
			return zed.Value{}, err
		}
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
		var err error
		r.typ, err = r.zctx.LookupTypeRecord(r.columns)
		if err != nil {
			return zed.Value{}, err
		}
	}
	return zed.Value{r.typ, b.Bytes()}, nil
}

type ArrayExpr struct {
	zctx    *zson.Context
	typ     *zed.TypeArray
	builder *zcode.Builder
	exprs   []Evaluator
}

func NewArrayExpr(zctx *zson.Context, exprs []Evaluator) *ArrayExpr {
	return &ArrayExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeArray(zed.TypeNull),
		builder: zcode.NewBuilder(),
		exprs:   exprs,
	}
}

func (a *ArrayExpr) Eval(rec *zed.Record) (zed.Value, error) {
	inner := a.typ.Type
	container := zed.IsContainerType(inner)
	b := a.builder
	b.Reset()
	var first zed.Type
	for _, e := range a.exprs {
		zv, err := e.Eval(rec)
		if err != nil {
			return zed.Value{}, err
		}
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
				return zed.NewErrorf("illegal mixed type array"), nil
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
	return zed.Value{a.typ, bytes}, nil
}

type SetExpr struct {
	zctx    *zson.Context
	typ     *zed.TypeSet
	builder *zcode.Builder
	exprs   []Evaluator
}

func NewSetExpr(zctx *zson.Context, exprs []Evaluator) *SetExpr {
	return &SetExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeSet(zed.TypeNull),
		builder: zcode.NewBuilder(),
		exprs:   exprs,
	}
}

func (s *SetExpr) Eval(rec *zed.Record) (zed.Value, error) {
	var inner zed.Type
	var container bool
	b := s.builder
	b.Reset()
	var first zed.Type
	for _, e := range s.exprs {
		zv, err := e.Eval(rec)
		if err != nil {
			return zed.Value{}, err
		}
		typ := zv.Type
		if first == nil {
			first = typ
		}
		if typ != inner && typ != zed.TypeNull {
			if typ == first || first == zed.TypeNull {
				s.typ = s.zctx.LookupTypeSet(zv.Type)
				inner = s.typ.Type
				container = zed.IsContainerType(inner)
			} else {
				return zed.NewErrorf("illegal mixed type array"), nil
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
		// Return empty set instead of null set.
		bytes = []byte{}
	}
	return zed.Value{s.typ, zed.NormalizeSet(bytes)}, nil
}

type Entry struct {
	Key Evaluator
	Val Evaluator
}

type MapExpr struct {
	zctx    *zson.Context
	typ     *zed.TypeMap
	builder *zcode.Builder
	entries []Entry
}

func NewMapExpr(zctx *zson.Context, entries []Entry) *MapExpr {
	return &MapExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeMap(zed.TypeNull, zed.TypeNull),
		builder: zcode.NewBuilder(),
		entries: entries,
	}
}

func (m *MapExpr) Eval(rec *zed.Record) (zed.Value, error) {
	var containerKey, containerVal bool
	var keyType, valType zed.Type
	b := m.builder
	b.Reset()
	for _, e := range m.entries {
		key, err := e.Key.Eval(rec)
		if err != nil {
			return zed.Value{}, err
		}
		val, err := e.Val.Eval(rec)
		if err != nil {
			return zed.Value{}, err
		}
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
			return zed.NewErrorf("illegal mixed type map"), nil
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
	return zed.Value{m.typ, zed.NormalizeMap(bytes)}, nil
}
