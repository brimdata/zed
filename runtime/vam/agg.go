package vam

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
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

func (c *CountByString) Pull(done bool) (vector.Any, error) {
	if done {
		_, err := c.parent.Pull(done)
		return nil, err
	}
	if c.done {
		return nil, nil
	}
	for {
		//XXX check context Done
		vec, err := c.parent.Pull(false)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			c.done = true
			return c.table.materialize(c.zctx, c.name), nil
		}
		if nulls, ok := vec.(*vector.Nulls); ok {
			vec = nulls.Values()
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
	if zed.TypeUnder(val.Type()) == zed.TypeString {
		c.table[zed.DecodeString(val.Bytes())] += uint64(vec.Length())
	}
}

func (c *countByString) materialize(zctx *zed.Context, name string) *vector.Record {
	typ := zctx.MustLookupTypeRecord([]zed.Field{
		{Type: zed.TypeString, Name: name},
		{Type: zed.TypeUint64, Name: "count"},
	})
	counts := make([]uint64, len(c.table))
	keys := make([]string, len(c.table))
	var off int
	for key, count := range c.table {
		keys[off] = key
		counts[off] = count
		off++
	}
	keyVec := vector.NewString(zed.TypeString, keys)
	countVec := vector.NewUint(zed.TypeUint64, counts)
	return vector.NewRecordWithFields(typ, []vector.Any{keyVec, countVec})
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

func (c *Sum) Pull(done bool) (vector.Any, error) {
	if done {
		_, err := c.parent.Pull(done)
		return nil, err
	}
	if c.done {
		return nil, nil
	}
	for {
		//XXX check context Done
		// XXX PullVec returns a single vector and enumerates through the
		// different underlying types that match a particular projection
		vec, err := c.parent.Pull(false)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			c.done = true
			return c.materialize(c.zctx, c.name), nil
		}
		if nulls, ok := vec.(*vector.Nulls); ok {
			vec = nulls.Values()
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

func (c *Sum) materialize(zctx *zed.Context, name string) *vector.Record {
	typ := zctx.MustLookupTypeRecord([]zed.Field{
		{Type: zed.TypeInt64, Name: "sum"},
	})
	return vector.NewRecordWithFields(typ, []vector.Any{vector.NewInt(zed.TypeInt64, []int64{c.sum})})
}
