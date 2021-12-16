package agg

import (
	"fmt"

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

func (u *Union) Consume(v zed.Value) {
	//XXX isnull
	if v.IsNil() {
		return
	}
	if u.typ == nil {
		u.typ = v.Type
	} else if u.typ != v.Type {
		//XXX we should make union type for the set-union
		// instead of silently ignoring
		return
	}
	u.update(v.Bytes)
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

func (u *Union) Result(zctx *zed.Context) zed.Value {
	if u.typ == nil {
		return zed.Null
	}
	var b zcode.Builder
	container := zed.IsContainerType(u.typ)
	for s := range u.val {
		if container {
			b.AppendContainer([]byte(s))
		} else {
			b.AppendPrimitive([]byte(s))
		}
	}
	setType := zctx.LookupTypeSet(u.typ)
	return zed.Value{setType, zed.NormalizeSet(b.Bytes())}
}

func (u *Union) ConsumeAsPartial(zv zed.Value) {
	if u.typ == nil {
		typ, ok := zv.Type.(*zed.TypeSet)
		if !ok {
			panic("union: partial not a set type")
		}
		u.typ = typ.Type
	}
	for it := zv.Iter(); !it.Done(); {
		elem, _, err := it.Next()
		if err != nil {
			panic(fmt.Errorf("union partial: set bytes are corrupt: %w", err))
		}
		u.update(elem)
	}
}

func (u *Union) ResultAsPartial(zctx *zed.Context) zed.Value {
	return u.Result(zctx)
}
