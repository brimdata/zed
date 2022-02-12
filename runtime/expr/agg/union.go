package agg

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Union struct {
	typ  zed.Type
	val  map[string]struct{}
	size int
}

var _ Function = (*Union)(nil)

func newUnion() *Union {
	return &Union{
		val: make(map[string]struct{}),
	}
}

func (u *Union) Consume(val *zed.Value) {
	if val.IsNull() {
		return
	}
	if u.typ == nil {
		u.typ = val.Type
	} else if u.typ != val.Type {
		// We should make union type for the set-union
		// instead of silently ignoring.  See #3363.
		return
	}
	u.update(val.Bytes)
}

func (u *Union) update(b zcode.Bytes) {
	if _, ok := u.val[string(b)]; !ok {
		u.val[string(b)] = struct{}{}
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
	for key := range u.val {
		u.size -= len(key)
		delete(u.val, key)
		return
	}
}

func (u *Union) Result(zctx *zed.Context) *zed.Value {
	if u.typ == nil {
		return zed.Null
	}
	var b zcode.Builder
	for v := range u.val {
		b.Append([]byte(v))
	}
	return zed.NewValue(zctx.LookupTypeSet(u.typ), zed.NormalizeSet(b.Bytes()))
}

func (u *Union) ConsumeAsPartial(val *zed.Value) {
	if u.typ == nil {
		typ, ok := val.Type.(*zed.TypeSet)
		if !ok {
			panic("union: partial not a set type")
		}
		u.typ = typ.Type
	}
	for it := val.Iter(); !it.Done(); {
		u.update(it.Next())
	}
}

func (u *Union) ResultAsPartial(zctx *zed.Context) *zed.Value {
	return u.Result(zctx)
}
