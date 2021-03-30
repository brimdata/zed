package agg

import (
	"errors"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
)

type Collect struct {
	typ  zng.Type
	val  []zcode.Bytes
	size int
}

func (c *Collect) Consume(v zng.Value) error {
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

func (c *Collect) Result(zctx *resolver.Context) (zng.Value, error) {
	if c.typ == nil {
		// no values found
		return zng.Value{Type: zng.TypeNull}, nil
	}
	var b zcode.Builder
	container := zng.IsContainerType(c.typ)
	for _, item := range c.val {
		if container {
			b.AppendContainer(item)
		} else {
			b.AppendPrimitive(item)
		}
	}
	typ := zctx.LookupTypeArray(c.typ)
	return zng.Value{typ, b.Bytes()}, nil
}

func (c *Collect) ConsumeAsPartial(zv zng.Value) error {
	if c.typ == nil {
		typ, ok := zv.Type.(*zng.TypeArray)
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

func (c *Collect) ResultAsPartial(tc *resolver.Context) (zng.Value, error) {
	return c.Result(tc)
}
