package expr

import (
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type RecordExpr struct {
	zctx    *zson.Context
	typ     *zng.TypeRecord
	builder *zcode.Builder
	columns []zng.Column
	exprs   []Evaluator
}

func NewRecordExpr(zctx *zson.Context, names []string, exprs []Evaluator) *RecordExpr {
	columns := make([]zng.Column, 0, len(names))
	for _, name := range names {
		columns = append(columns, zng.Column{Name: name})
	}
	return &RecordExpr{
		zctx:    zctx,
		builder: zcode.NewBuilder(),
		columns: columns,
		exprs:   exprs,
	}
}

func (r *RecordExpr) Eval(rec *zng.Record) (zng.Value, error) {
	var changed bool
	b := r.builder
	b.Reset()
	for k, e := range r.exprs {
		zv, err := e.Eval(rec)
		if err != nil {
			return zng.Value{}, err
		}
		if r.columns[k].Type != zv.Type {
			r.columns[k].Type = zv.Type
			changed = true
		}
		if zng.IsContainerType(zv.Type) {
			b.AppendContainer(zv.Bytes)
		} else {
			b.AppendPrimitive(zv.Bytes)
		}
	}
	if changed {
		var err error
		r.typ, err = r.zctx.LookupTypeRecord(r.columns)
		if err != nil {
			return zng.Value{}, err
		}
	}
	return zng.Value{r.typ, b.Bytes()}, nil
}

type ArrayExpr struct {
	zctx    *zson.Context
	typ     *zng.TypeArray
	builder *zcode.Builder
	exprs   []Evaluator
}

func NewArrayExpr(zctx *zson.Context, exprs []Evaluator) *ArrayExpr {
	return &ArrayExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeArray(zng.TypeNull),
		builder: zcode.NewBuilder(),
		exprs:   exprs,
	}
}

func (a *ArrayExpr) Eval(rec *zng.Record) (zng.Value, error) {
	inner := a.typ.Type
	container := zng.IsContainerType(inner)
	b := a.builder
	b.Reset()
	var first zng.Type
	for _, e := range a.exprs {
		zv, err := e.Eval(rec)
		if err != nil {
			return zng.Value{}, err
		}
		typ := zv.Type
		if first == nil {
			first = typ
		}
		if typ != inner && typ != zng.TypeNull {
			if typ == first || first == zng.TypeNull {
				a.typ = a.zctx.LookupTypeArray(zv.Type)
				inner = a.typ.Type
				container = zng.IsContainerType(inner)
			} else {
				return zng.NewErrorf("illegal mixed type array"), nil
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
	return zng.Value{a.typ, bytes}, nil
}

type SetExpr struct {
	zctx    *zson.Context
	typ     *zng.TypeSet
	builder *zcode.Builder
	exprs   []Evaluator
}

func NewSetExpr(zctx *zson.Context, exprs []Evaluator) *SetExpr {
	return &SetExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeSet(zng.TypeNull),
		builder: zcode.NewBuilder(),
		exprs:   exprs,
	}
}

func (s *SetExpr) Eval(rec *zng.Record) (zng.Value, error) {
	var inner zng.Type
	var container bool
	b := s.builder
	b.Reset()
	var first zng.Type
	for _, e := range s.exprs {
		zv, err := e.Eval(rec)
		if err != nil {
			return zng.Value{}, err
		}
		typ := zv.Type
		if first == nil {
			first = typ
		}
		if typ != inner && typ != zng.TypeNull {
			if typ == first || first == zng.TypeNull {
				s.typ = s.zctx.LookupTypeSet(zv.Type)
				inner = s.typ.Type
				container = zng.IsContainerType(inner)
			} else {
				return zng.NewErrorf("illegal mixed type array"), nil
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
	return zng.Value{s.typ, zng.NormalizeSet(bytes)}, nil
}

type Entry struct {
	Key Evaluator
	Val Evaluator
}

type MapExpr struct {
	zctx    *zson.Context
	typ     *zng.TypeMap
	builder *zcode.Builder
	entries []Entry
}

func NewMapExpr(zctx *zson.Context, entries []Entry) *MapExpr {
	return &MapExpr{
		zctx:    zctx,
		typ:     zctx.LookupTypeMap(zng.TypeNull, zng.TypeNull),
		builder: zcode.NewBuilder(),
		entries: entries,
	}
}

func (m *MapExpr) Eval(rec *zng.Record) (zng.Value, error) {
	var containerKey, containerVal bool
	var keyType, valType zng.Type
	b := m.builder
	b.Reset()
	for _, e := range m.entries {
		key, err := e.Key.Eval(rec)
		if err != nil {
			return zng.Value{}, err
		}
		val, err := e.Val.Eval(rec)
		if err != nil {
			return zng.Value{}, err
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
			containerKey = zng.IsContainerType(keyType)
			containerVal = zng.IsContainerType(valType)
		} else if keyType != m.typ.KeyType || valType != m.typ.ValType {
			return zng.NewErrorf("illegal mixed type map"), nil
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
	return zng.Value{m.typ, zng.NormalizeMap(bytes)}, nil
}
