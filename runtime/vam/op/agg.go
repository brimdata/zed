package op

import (
	"fmt"
	"slices"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/vector"
)

//XXX need to make sure vam operator objects are returned to GC as they are finished

type CountByString struct {
	parent vector.Puller
	zctx   *zed.Context
	field  expr.Evaluator
	name   string
	table  countByString
	done   bool
}

func NewCountByString(zctx *zed.Context, parent vector.Puller, name string) *CountByString {
	return &CountByString{
		parent: parent,
		zctx:   zctx,
		field:  expr.NewDotExpr(zctx, &expr.This{}, name),
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
		c.update(vec)
	}
}

func (c *CountByString) update(val vector.Any) {
	if val, ok := val.(*vector.Variant); ok {
		for _, val := range val.Values {
			c.update(val)
		}
		return
	}
	switch val := val.(type) {
	case *vector.Dict:
		s, ok := val.Any.(*vector.String)
		if !ok {
			panic(fmt.Sprintf("UNKNOWN %T\n", val))
		}
		c.table.countDict(s, val.Index, val.Counts)
	case *vector.String:
		c.table.count(val)
	case *vector.Const:
		c.table.countFixed(val)
	default:
		panic(fmt.Sprintf("UNKNOWN %T\n", val))
	}
}

type countByString struct {
	nulls uint64
	table map[string]uint64
}

//XXX how to tell the difference between a view as a selection and a view as
// an index into a dict... maybe just keep Dict separate and have the counts in the Dict,
// but we still need the view on the dict to expand things...

func (c *countByString) count(vec *vector.String) {
	offs := vec.Offs
	bytes := vec.Bytes
	for k := range offs {
		c.table[string(bytes[offs[k]:offs[k+1]])]++
	}
}

func (c *countByString) countDict(val *vector.String, idx []byte, counts []uint32) {
	offs := val.Offs
	bytes := val.Bytes
	n := uint32(len(offs) - 1)
	for tag := uint32(0); tag < n; tag++ {
		c.table[string(bytes[offs[idx[tag]]:offs[idx[tag]+1]])] += uint64(counts[tag])
	}
}

func (c *countByString) countFixed(vec *vector.Const) {
	//XXX
	val := vec.Value()
	switch val.Type().ID() {
	case zed.IDString:
		c.table[zed.DecodeString(val.Bytes())] += uint64(vec.Length())
	case zed.IDNull:
		c.nulls += uint64(vec.Length())
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
		bytes = append(bytes, key...)
		counts[k] = count
		k++
	}
	offs[k] = uint32(len(bytes))
	// XXX change nulls to string null... this will be fixed in
	// prod-quality summarize op
	var nulls *vector.Bool
	if c.nulls > 0 {
		length++
		counts = slices.Grow(counts, length)[0:length]
		offs = slices.Grow(offs, length+1)[0 : length+1]
		counts[k] = c.nulls
		k++
		offs[k] = uint32(len(bytes))
		nulls = vector.NewBoolEmpty(uint32(k), nil)
		nulls.Set(uint32(k - 1))
	}
	keyVec := vector.NewString(offs, bytes, nulls)
	countVec := vector.NewUint(zed.TypeUint64, counts, nil)
	return vector.NewRecord(typ, []vector.Any{keyVec, countVec}, uint32(length), nil)
}

type Sum struct {
	parent vector.Puller
	zctx   *zed.Context
	field  expr.Evaluator
	name   string
	sum    int64
	done   bool
}

func NewSum(zctx *zed.Context, parent vector.Puller, name string) *Sum {
	return &Sum{
		parent: parent,
		zctx:   zctx,
		field:  expr.NewDotExpr(zctx, &expr.This{}, name),
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
	switch vec := c.field.Eval(vec).(type) {
	case *vector.Int:
		for _, x := range vec.Values {
			c.sum += x
		}
	case *vector.Uint:
		for _, x := range vec.Values {
			c.sum += int64(x)
		}
	case *vector.Dict:
		switch number := vec.Any.(type) {
		case *vector.Int:
			for k, val := range number.Values {
				c.sum += val * int64(vec.Counts[k])
			}
		case *vector.Uint:
			for k, val := range number.Values {
				c.sum += int64(val) * int64(vec.Counts[k])
			}
		}
	}
}

func (c *Sum) materialize(zctx *zed.Context, name string) *vector.Record {
	typ := zctx.MustLookupTypeRecord([]zed.Field{
		{Type: zed.TypeInt64, Name: name},
	})
	return vector.NewRecord(typ, []vector.Any{vector.NewInt(zed.TypeInt64, []int64{c.sum}, nil)}, 1, nil)
}
