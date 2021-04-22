package agg

import (
	"errors"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Union struct {
	typ  zng.Type
	val  map[string]struct{}
	size int
}

func newUnion() *Union {
	return &Union{
		val: make(map[string]struct{}),
	}
}

func (u *Union) Consume(v zng.Value) error {
	if v.IsNil() {
		return nil
	}
	if u.typ == nil {
		u.typ = v.Type
	} else if u.typ != v.Type {
		//u.TypeMismatch++
		return nil
	}
	return u.update(v.Bytes)
}

func (u *Union) update(b zcode.Bytes) error {
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
	return nil
}

func (u *Union) deleteOne() {
	for key := range u.val {
		u.size -= len(key)
		delete(u.val, key)
		return
	}
}

func (u *Union) Result(zctx *zson.Context) (zng.Value, error) {
	if u.typ == nil {
		return zng.Value{Type: zng.TypeNull}, nil
	}
	var b zcode.Builder
	container := zng.IsContainerType(u.typ)
	for s := range u.val {
		if container {
			b.AppendContainer([]byte(s))
		} else {
			b.AppendPrimitive([]byte(s))
		}
	}
	setType := zctx.LookupTypeSet(u.typ)
	return zng.Value{setType, zng.NormalizeSet(b.Bytes())}, nil
}

func (u *Union) ConsumeAsPartial(zv zng.Value) error {
	if u.typ == nil {
		typ, ok := zv.Type.(*zng.TypeSet)
		if !ok {
			return errors.New("partial not a set type")
		}
		u.typ = typ.Type
	}
	for it := zv.Iter(); !it.Done(); {
		elem, _, err := it.Next()
		if err != nil {
			return err
		}
		if err := u.update(elem); err != nil {
			return err
		}
	}
	return nil
}

func (u *Union) ResultAsPartial(zctx *zson.Context) (zng.Value, error) {
	return u.Result(zctx)
}
