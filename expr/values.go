package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
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

type RecordExprWith struct {
	zctx     *zed.Context
	with     Evaluator
	names    []string
	exprs    []Evaluator
	position map[string]int
	builder  zcode.Builder
	columns  []zed.Column
	cache    *zed.TypeRecord
}

func NewRecordExprWith(zctx *zed.Context, names []string, exprs []Evaluator, with Evaluator) (*RecordExprWith, error) {
	assignments := make([]Assignment, 0, len(names))
	position := make(map[string]int)
	for k, name := range names {
		position[name] = k
		assignments = append(assignments, Assignment{
			LHS: field.New(name),
			RHS: exprs[k],
		})
	}
	return &RecordExprWith{
		zctx:     zctx,
		with:     with,
		names:    names,
		exprs:    exprs,
		position: position,
	}, nil
}

func (r *RecordExprWith) Eval(ectx Context, this *zed.Value) *zed.Value {
	with := r.with.Eval(ectx, this)
	if with.IsMissing() {
		return with
	}
	typ := zed.TypeRecordOf(with.Type)
	if typ == nil {
		return r.zctx.Missing()
	}
	// cols is a cache of the columns of the output record.
	// As long as the type doesn't change, the columns will stay the
	// same and the dirty boolean will stay false.  If something is
	// diferrent, dirty becomes true, we look up the new type, and
	// it stays in the cache for the next input.
	cols := r.columns
	b := r.builder
	b.Reset()
	var dirty bool
	it := with.Iter()
	for k, col := range typ.Columns {
		if pos, ok := r.position[col.Name]; ok {
			val := r.exprs[pos].Eval(ectx, this)
			b.Append(val.Bytes)
			col.Type = val.Type
			it.Next()
		} else {
			b.Append(it.Next())
		}
		if k >= len(cols) {
			dirty = true
			cols = append(cols, col)
		} else if col != cols[k] {
			dirty = true
			cols[k] = col
		}
	}
	colno := len(typ.Columns)
	for k, e := range r.exprs {
		if typ.HasField(r.names[k]) {
			continue
		}
		val := e.Eval(ectx, this)
		b.Append(val.Bytes)
		col := zed.Column{r.names[k], val.Type}
		if colno >= len(cols) {
			dirty = true
			cols = append(cols, col)
		} else if col != cols[colno] {
			dirty = true
			cols[colno] = col
		}
		colno++
	}
	if colno < len(cols) {
		dirty = true
		cols = cols[:colno]
	}
	if dirty {
		r.cache = r.zctx.MustLookupTypeRecord(cols)
		r.columns = cols
	}
	return ectx.NewValue(r.cache, b.Bytes())
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
