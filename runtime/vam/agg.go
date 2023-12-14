package vam

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// XXX use zed.Value for slow path stuff, e.g., when the group-by key is
// a complex type.  when we improve the zed.Value impl this will get better.

// one aggTable per fixed set of types of aggs and keys.
type aggTable interface {
	update([]vector.Any, []vector.Any)
	materialize() vector.Any
} //XXX

// XXX sum, min, max, etc forming one agg column of an aggTable.
// AggFunc is a template that generates a new monad for each row
// (i.e., unique set of keys).
type AggFunc interface {
	updateInt(int64)
	updateUint(uint64)
	updateFloat(float64)
	updateBool(bool)
}

//XXX need to make sure vam operator objects are returned to GC as they are finished

type superTable struct {
	keyBytes []byte
	table    map[string]aggRow // dont'export
	patterns []AggPattern      //XXX don't export
}

var _ aggTable = (*superTable)(nil)

type aggRow struct {
	keys  []*zed.Value
	funcs []AggFunc
}

func (s *superTable) update(keys []vector.Any, args []vector.Any) {
	//XXX vectors might be nil?
	nslot := args[0].Length()
	for slot := 0; slot < nslot; slot++ {
		s.keyBytes = s.keyBytes[:0]
		for _, key := range keys {
			s.keyBytes = key.Key(s.keyBytes, slot)
		}
		row, ok := s.table[string(s.keyBytes)]
		if !ok {
			aggFuncs := make([]AggFunc, len(s.patterns))
			for k, pattern := range s.patterns {
				aggFuncs[k] = pattern()
			}
			keyVals := make([]*zed.Value, 0, len(keys))
			for _, key := range keys {
				keyVals = append(keyVals, key.Serialize(slot))
			}
			s.table[string(s.keyBytes)] = aggRow{keys: keyVals, funcs: aggFuncs}
		}
		for k, agg := range row.funcs {
			if arg := args[k]; arg != nil {
				//XXX type under might not be right for unions
				switch arg := arg.(type) {
				case *vector.Int:
					agg.updateInt(arg.Values[slot])
				case *vector.Uint:
					agg.updateUint(arg.Values[slot])
				case *vector.Float:
					agg.updateFloat(arg.Values[slot])
				case *vector.Bool:
					agg.updateBool(arg.Values[slot])
				default:
					// XXX do something for skipped types?
				}
			}
		}
	}
}

func (s *superTable) materialize() vector.Any {
	return nil //XXX
}

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

type AggPattern func() AggFunc

func NewAggPattern(op string, hasarg bool) (AggPattern, error) {
	needarg := true
	var pattern AggPattern
	switch op {
	case "count":
		needarg = false
		pattern = func() AggFunc {
			return newAggCount()
		}
	/* not yet
	case "any":
		pattern = func() AggFunc {
			return &Any{}
		}
	case "avg":
		pattern = func() AggFunc {
			return &Avg{}
		}
	case "dcount":
		pattern = func() AggFunc {
			return NewDCount()
		}
	case "fuse":
		pattern = func() AggFunc {
			return newFuse()
		}
	*/
	case "sum":
		pattern = func() AggFunc {
			return newAggSum()
		}
	/* not yet
	case "min":
		pattern = func() AggFunc {
			return newMathReducer(anymath.Min)
		}
	case "max":
		pattern = func() AggFunc {
			return newMathReducer(anymath.Max)
		}
	case "union":
		panic("TBD")
	case "collect":
		panic("TBD")
	case "and":
		pattern = func() AggFunc {
			return &aggAnd{}
		}
	case "or":
		pattern = func() AggFunc {
			return &aggOr{}
		}
	*/
	default:
		return nil, fmt.Errorf("unknown aggregation function: %s", op)
	}
	if needarg && !hasarg {
		return nil, fmt.Errorf("%s: argument required", op)
	}
	return pattern, nil
}

type aggSum struct {
	typeID int
	word   uint64
}

func newAggSum() *aggSum {
	return &aggSum{
		typeID: zed.IDInt64,
	}
}

func (a *aggSum) updateInt(v int64) {
	if a.typeID == zed.IDInt64 {
		*((*int64)(unsafe.Pointer(&a.word))) += v
		return
	}
	*((*float64)(unsafe.Pointer(&a.word))) += float64(v)
}

func (a *aggSum) updateUint(v uint64) {
	//XXX fix rest of logic too
	panic("TBD")
}

func (a *aggSum) updateFloat(v float64) {
	if a.typeID == zed.IDInt64 {
		a.typeID = zed.IDFloat64
		old := float64(*((*int64)(unsafe.Pointer(&a.word))))
		*((*float64)(unsafe.Pointer(&a.word))) = old + v
		return
	}
	*((*float64)(unsafe.Pointer(&a.word))) += float64(v)
}

func (a *aggSum) updateBool(bool) {}

type aggCount struct {
	count uint64
}

func newAggCount() *aggCount {
	return &aggCount{}
}

func (a *aggCount) updateInt(int64) {
	a.count++
}

func (a *aggCount) updateBool(bool) {
	a.count++
}

func (a *aggCount) updateUint(uint64) {
	a.count++
}

func (a *aggCount) updateFloat(float64) {
	a.count++
}
