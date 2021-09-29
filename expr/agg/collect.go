package agg

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Collect struct {
	typ  zed.Type
	val  []zcode.Bytes
	size int
}

func (c *Collect) Consume(v zed.Value) error {
	if v.IsNil() {
		return nil
	}
	if c.typ == nil {
		c.typ = v.Type
	} else if c.typ != v.Type {
		//c.TypeMismatch++
		return nil
	}
	c.update(v.Bytes)
	return nil
}

func (c *Collect) update(b zcode.Bytes) {
	stash := make(zcode.Bytes, len(b))
	copy(stash, b)
	c.val = append(c.val, stash)
	c.size += len(b)
	for c.size > MaxValueSize {
		// XXX See issue #1813.  For now we silently discard entries
		// to maintain the size limit.
		//c.MemExceeded++
		c.size -= len(c.val[0])
		c.val = c.val[1:]
	}
}

func (c *Collect) Result(zctx *zson.Context) (zed.Value, error) {
	if c.typ == nil {
		// no values found
		return zed.Value{Type: zed.TypeNull}, nil
	}
	var b zcode.Builder
	container := zed.IsContainerType(c.typ)
	for _, item := range c.val {
		if container {
			b.AppendContainer(item)
		} else {
			b.AppendPrimitive(item)
		}
	}
	typ := zctx.LookupTypeArray(c.typ)
	return zed.Value{typ, b.Bytes()}, nil
}

func (c *Collect) ConsumeAsPartial(zv zed.Value) error {
	if c.typ == nil {
		typ, ok := zv.Type.(*zed.TypeArray)
		if !ok {
			return errors.New("partial not an array type")
		}
		c.typ = typ.Type
	}
	for it := zv.Iter(); !it.Done(); {
		elem, _, err := it.Next()
		if err != nil {
			return err
		}
		c.update(elem)
	}
	return nil
}

func (c *Collect) ResultAsPartial(tc *zson.Context) (zed.Value, error) {
	return c.Result(tc)
}
