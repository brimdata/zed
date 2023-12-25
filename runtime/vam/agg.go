package vam

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

//XXX need to make sure vam operator objects are returned to GC as they are finished

type CountByString struct {
	parent Puller
	zctx   *zed.Context
	field  Evaluator
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
		field:  NewDotExpr(zctx, &This{}, name),
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
		c.update(vec)
	}
}

func (c *CountByString) update(vec vector.Any) {
	if vec, ok := vec.(*vector.Variant); ok {
		for _, vec := range vec.Values {
			c.update(vec)
		}
		return
	}
	vec = c.field.Eval(vec)
	switch vec := vec.(type) {
	case *vector.String:
		c.table.count(vec)
	case *vector.DictString:
		c.table.countDict(vec)
	case (*vector.Const):
		c.table.countFixed(vec)
	}
}

type countByString struct {
	table map[string]uint64
}

func (c *countByString) count(vec *vector.String) {
	offs := vec.Offsets
	bytes := vec.Bytes
	for k := range offs {
		c.table[string(bytes[offs[k]:offs[k+1]])] += 1
	}
}

func (c *countByString) countDict(vec *vector.DictString) {
	offs := vec.Offs
	bytes := vec.Bytes
	for k := range offs {
		tag := vec.Tags[k]
		c.table[string(bytes[offs[tag]:offs[tag+1]])] += uint64(vec.Counts[tag])
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
	length := len(c.table)
	counts := make([]uint64, length)
	var bytes []byte
	offs := make([]uint32, length+1)
	var k int
	for key, count := range c.table {
		offs[k] = uint32(len(bytes))
		bytes = append(bytes, []byte(key)...)
		counts[k] = count
		k++
	}
	offs[k] = uint32(len(bytes))
	keyVec := vector.NewString(offs, bytes, nil)
	countVec := vector.NewUint(zed.TypeUint64, counts, nil)
	return vector.NewRecord(typ, []vector.Any{keyVec, countVec}, uint32(length), nil)
}

type Sum struct {
	parent Puller
	zctx   *zed.Context
	field  Evaluator
	name   string
	sum    int64
	done   bool
}

func NewSum(zctx *zed.Context, parent Puller, name string) *Sum {
	return &Sum{
		parent: parent,
		zctx:   zctx,
		name:   name,
		field:  NewDotExpr(zctx, &This{}, name),
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
		c.update(vec)
	}
}

func (c *Sum) update(vec vector.Any) {
	if vec, ok := vec.(*vector.Variant); ok {
		for _, vec := range vec.Values {
			c.update(vec)
		}
		return
	}
	vec = c.field.Eval(vec)
	switch vec := vec.(type) {
	case *vector.Int:
		for _, x := range vec.Values {
			c.sum += x
		}
	case *vector.DictInt:
		for k := range vec.Values {
			c.sum += vec.Values[k] * int64(vec.Counts[k])
		}
	case *vector.Uint:
		for _, x := range vec.Values {
			c.sum += int64(x)
		}
	case *vector.DictUint:
		for k := range vec.Values {
			c.sum += int64(vec.Values[k]) * int64(vec.Counts[k])
		}
	}
}

func (c *Sum) materialize(zctx *zed.Context, name string) *vector.Record {
	typ := zctx.MustLookupTypeRecord([]zed.Field{
		{Type: zed.TypeInt64, Name: "sum"},
	})
	return vector.NewRecord(typ, []vector.Any{vector.NewInt(zed.TypeInt64, []int64{c.sum}, nil)}, 1, nil)
}
