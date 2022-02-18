package agg

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Union struct {
	types map[zed.Type]map[string]struct{}
	size  int
}

var _ Function = (*Union)(nil)

func newUnion() *Union {
	return &Union{
		types: make(map[zed.Type]map[string]struct{}),
	}
}

func (u *Union) Consume(val *zed.Value) {
	if val.IsNull() {
		return
	}
	u.update(val.Type, val.Bytes)
}

func (u *Union) update(typ zed.Type, b zcode.Bytes) {
	m, ok := u.types[typ]
	if !ok {
		m = make(map[string]struct{})
		u.types[typ] = m
	}
	if _, ok := m[string(b)]; !ok {
		m[string(b)] = struct{}{}
		u.size += len(b)
		for u.size > MaxValueSize {
			u.deleteOne()
			// XXX See issue #1813.  For now, we silently discard
			// entries to maintain the size limit.
			//return ErrRowTooBig
		}
	}
}

func (u *Union) deleteOne() {
	for typ, m := range u.types {
		for key := range m {
			u.size -= len(key)
			delete(m, key)
			if len(m) == 0 {
				delete(u.types, typ)
			}
			return
		}
	}
}

func (u *Union) Result(zctx *zed.Context) *zed.Value {
	if len(u.types) == 0 {
		return zed.Null
	}
	types := make([]zed.Type, 0, len(u.types))
	for typ := range u.types {
		types = append(types, typ)
	}
	types = zed.CanonicalUnionOfTypes(types)
	inner := types[0]
	if len(types) > 1 {
		inner = zctx.LookupTypeUnion(types)
	}
	var b zcode.Builder
	for typ, m := range u.types {
		for v := range m {
			if union, ok := zed.TypeUnder(inner).(*zed.TypeUnion); ok {
				zed.BuildUnion(&b, union.Selector(typ), []byte(v))
			} else {
				b.Append([]byte(v))
			}
		}
	}
	return zed.NewValue(zctx.LookupTypeSet(inner), zed.NormalizeSet(b.Bytes()))
}

func (u *Union) ConsumeAsPartial(val *zed.Value) {
	if val.IsNull() {
		return
	}
	styp, ok := val.Type.(*zed.TypeSet)
	if !ok {
		panic("union: partial not a set type")
	}
	for it := val.Iter(); !it.Done(); {
		typ := styp.Type
		b := it.Next()
		if union, ok := zed.TypeUnder(typ).(*zed.TypeUnion); ok {
			typ, _, b = union.SplitZNG(b)
		}
		u.update(typ, b)
	}
}

func (u *Union) ResultAsPartial(zctx *zed.Context) *zed.Value {
	return u.Result(zctx)
}
