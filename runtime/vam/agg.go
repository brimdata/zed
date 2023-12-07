package vam

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

//XXX need to make sure vam operator objects are returned to GC as they are finished

type CountByString struct {
	parent Puller
	zctx   *zed.Context
	name   string
	table  countByString
	done   bool
}

func NewCountByString(zctx *zed.Context, parent Puller, name string) *CountByString {
	return &CountByString{
		parent: parent,
		zctx:   zctx,
		name:   name,
		table:  countByString{table: make(map[string]uint64)}, //XXX
	}
}

func (*CountByString) PullVec(_ bool) (vector.Any, error) {
	return nil, errors.New("internal error: vam.CountByString.PullVec called")
}

func (c *CountByString) Pull(done bool) (zbuf.Batch, error) {
	if done {
		_, err := c.parent.PullVec(done)
		return nil, err
	}
	if c.done {
		return nil, nil
	}
	for {
		//XXX check context Done
		vec, err := c.parent.PullVec(false)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			c.done = true
			return c.table.materialize(c.zctx, c.name), nil
		}
		if vec, ok := vec.(*vector.String); ok {
			c.table.count(vec)
			continue
		}
		if vec, ok := vec.(*vector.Const); ok {
			c.table.countFixed(vec)
			continue
		}
		//xxx
		fmt.Printf("vector.CountByString: bad vec %s %T\n", zson.String(vec.Type()), vec)
	}
}

/*
// XXX Let's use Pull() here... read whole column into Batch for better perf
func (c *CountByString) AsReader() zio.Reader {
	cbs := countByString{make(map[string]uint64)}
	for _, vec := range c.vecs {
		cbs.count(vec)
	}
	return cbs.materialize(c.zctx, c.name)
}
*/

type countByString struct {
	table map[string]uint64
}

func (c *countByString) count(vec *vector.String) {
	for _, s := range vec.Values {
		c.table[s] += 1
	}
}

func (c *countByString) countFixed(vec *vector.Const) {
	//XXX
	val := vec.Value()
	if zed.TypeUnder(val.Type) == zed.TypeString {
		c.table[zed.DecodeString(val.Bytes())] += uint64(vec.Length())
	}
}

func (c *countByString) materialize(zctx *zed.Context, name string) *zbuf.Array {
	typ := zctx.MustLookupTypeRecord([]zed.Field{
		{Type: zed.TypeString, Name: name},
		{Type: zed.TypeUint64, Name: "count"},
	})
	var b zcode.Builder
	vals := make([]zed.Value, len(c.table))
	var off int
	for key, count := range c.table {
		b.Reset()
		b.BeginContainer()
		b.Append(zed.EncodeString(key))
		b.Append(zed.EncodeUint(count))
		b.EndContainer()
		vals[off] = *zed.NewValue(typ, b.Bytes().Body())
		off++
	}
	return zbuf.NewArray(vals)
}

type Sum struct {
	parent Puller
	zctx   *zed.Context
	name   string
	sum    int64
	done   bool
}

func NewSum(zctx *zed.Context, parent Puller, name string) *Sum {
	return &Sum{
		parent: parent,
		zctx:   zctx,
		name:   name,
	}
}

func (*Sum) PullVec(_ bool) (vector.Any, error) {
	return nil, errors.New("internal error: vam.Sum.PullVec called")
}

func (c *Sum) Pull(done bool) (zbuf.Batch, error) {
	if done {
		_, err := c.parent.PullVec(done)
		return nil, err
	}
	if c.done {
		return nil, nil
	}
	for {
		//XXX check context Done
		// XXX PullVec returns a single vector and enumerates through the
		// different underlying types that match a particular projection
		vec, err := c.parent.PullVec(false)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			c.done = true
			return c.materialize(c.zctx, c.name), nil
		}
		if vec, ok := vec.(*vector.Int); ok {
			for _, x := range vec.Values {
				c.sum += x
			}
		}
		if vec, ok := vec.(*vector.Uint); ok {
			for _, x := range vec.Values {
				c.sum += int64(x)
			}
		}
	}
}

func (c *Sum) materialize(zctx *zed.Context, name string) *zbuf.Array {
	typ := zctx.MustLookupTypeRecord([]zed.Field{
		{Type: zed.TypeInt64, Name: "sum"},
	})
	var b zcode.Builder
	b.BeginContainer()
	b.Append(zed.EncodeInt(c.sum))
	b.EndContainer()
	return zbuf.NewArray([]zed.Value{*zed.NewValue(typ, b.Bytes().Body())})
}
