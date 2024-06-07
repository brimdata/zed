package expr

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zio"
)

func (c *Comparator) sortStableIndices(vals []zed.Value) []uint32 {
	if len(c.exprs) == 0 {
		return nil
	}
	n := len(vals)
	if max := math.MaxUint32; n > max {
		panic(fmt.Sprintf("number of values exceeds %d", max))
	}
	indices := make([]uint32, n)
	i64s := make([]int64, n)
	val0s := make([]zed.Value, n)
	arena := zed.NewArena()
	defer arena.Unref()
	ectx := NewContext(arena)
	native := true
	for i := range indices {
		indices[i] = uint32(i)
		val := c.exprs[0].Eval(ectx, vals[i])
		val0s[i] = val
		if id := val.Type().ID(); id <= zed.IDTime {
			if val.IsNull() {
				if c.nullsMax {
					i64s[i] = math.MaxInt64
				} else {
					i64s[i] = math.MinInt64
				}
			} else if zed.IsSigned(id) {
				i64s[i] = val.Int()
			} else {
				v := val.Uint()
				if v > math.MaxInt64 {
					v = math.MaxInt64
				}
				i64s[i] = int64(v)
			}
		} else {
			native = false
		}
	}
	arena = zed.NewArena()
	defer arena.Unref()
	ectx = NewContext(arena)
	sort.SliceStable(indices, func(i, j int) bool {
		if c.reverse {
			i, j = j, i
		}
		iidx, jidx := indices[i], indices[j]
		for k, expr := range c.exprs {
			arena.Reset()
			var ival, jval zed.Value
			if k == 0 {
				if native {
					if i64, j64 := i64s[iidx], i64s[jidx]; i64 != j64 {
						return i64 < j64
					} else if i64 != math.MaxInt64 && i64 != math.MinInt64 {
						continue
					}
				}
				ival, jval = val0s[iidx], val0s[jidx]
			} else {
				ival = expr.Eval(ectx, vals[iidx])
				jval = expr.Eval(ectx, vals[jidx])
			}
			if v := compareValues(arena, ival, jval, c.nullsMax); v != 0 {
				return v < 0
			}
		}
		return false
	})
	return indices
}

type CompareFn func(a, b zed.Value) int

// NewCompareFn creates a function that compares two values a and b according to
// nullsMax and exprs.  To compare a and b, it iterates over the elements e of
// exprs, stopping when e(a)!=e(b).  The handling of missing and null
// (collectively refered to as "null") values is governed by nullsMax.  If
// nullsMax is true, a null value is considered larger than any non-null value,
// and vice versa.
func NewCompareFn(nullsMax bool, exprs ...Evaluator) CompareFn {
	return NewComparator(nullsMax, false, exprs...).WithMissingAsNull().Compare
}

func NewValueCompareFn(o order.Which, nullsMax bool) CompareFn {
	return NewComparator(nullsMax, o == order.Desc, &This{}).Compare
}

type Comparator struct {
	ectx     Context
	exprs    []Evaluator
	nullsMax bool
	reverse  bool
}

// NewComparator returns a zed.Value comparator for exprs according to nullsMax
// and reverse.  To compare values a and b, it iterates over the elements e of
// exprs, stopping when e(a)!=e(b).  nullsMax determines whether a null value
// compares larger (if true) or smaller (if false) than a non-null value.
// reverse reverses the sense of comparisons.
func NewComparator(nullsMax, reverse bool, exprs ...Evaluator) *Comparator {
	return &Comparator{
		ectx:     NewContext(zed.NewArena()),
		exprs:    slices.Clone(exprs),
		nullsMax: nullsMax,
		reverse:  reverse,
	}
}

// WithMissingAsNull returns the receiver after modifying it to treat missing
// values as the null value in comparisons.
func (c *Comparator) WithMissingAsNull() *Comparator {
	for i, k := range c.exprs {
		c.exprs[i] = &missingAsNull{k}
	}
	return c
}

type missingAsNull struct{ Evaluator }

func (m *missingAsNull) Eval(ectx Context, val zed.Value) zed.Value {
	val = m.Evaluator.Eval(ectx, val)
	if val.IsMissing() {
		return zed.Null
	}
	return val
}

// Compare returns an interger comparing two values according to the receiver's
// configuration.  The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func (c *Comparator) Compare(a, b zed.Value) int {
	if c.reverse {
		a, b = b, a
	}
	for _, k := range c.exprs {
		c.ectx.Arena().Reset()
		aval := k.Eval(c.ectx, a)
		bval := k.Eval(c.ectx, b)
		if v := compareValues(c.ectx.Arena(), aval, bval, c.nullsMax); v != 0 {
			return v
		}
	}
	return 0
}

func compareValues(arena *zed.Arena, a, b zed.Value, nullsMax bool) int {
	// Handle nulls according to nullsMax
	nullA := a.IsNull()
	nullB := b.IsNull()
	if nullA && nullB {
		return 0
	}
	if nullA {
		if nullsMax {
			return 1
		} else {
			return -1
		}
	}
	if nullB {
		if nullsMax {
			return -1
		} else {
			return 1
		}
	}
	switch aid, bid := a.Type().ID(), b.Type().ID(); {
	case zed.IsNumber(aid) && zed.IsNumber(bid):
		return compareNumbers(a, b, aid, bid)
	case aid != bid:
		return zed.CompareTypes(a.Type(), b.Type())
	case aid == zed.IDBool:
		if av, bv := a.Bool(), b.Bool(); av == bv {
			return 0
		} else if av {
			return 1
		}
		return -1
	case aid == zed.IDBytes:
		return bytes.Compare(zed.DecodeBytes(a.Bytes()), zed.DecodeBytes(b.Bytes()))
	case aid == zed.IDString:
		return cmp.Compare(zed.DecodeString(a.Bytes()), zed.DecodeString(b.Bytes()))
	case aid == zed.IDIP:
		return zed.DecodeIP(a.Bytes()).Compare(zed.DecodeIP(b.Bytes()))
	case aid == zed.IDType:
		zctx := zed.NewContext() // XXX This is expensive.
		// XXX This isn't cheap eventually we should add
		// zed.CompareTypeValues(a, b zcode.Bytes).
		av, _ := zctx.DecodeTypeValue(a.Bytes())
		bv, _ := zctx.DecodeTypeValue(b.Bytes())
		return zed.CompareTypes(av, bv)
	}
	// XXX record support easy to add here if we moved the creation of the
	// field resolvers into this package.
	if innerType := zed.InnerType(a.Type()); innerType != nil {
		ait, bit := a.Iter(), b.Iter()
		for {
			if ait.Done() {
				if bit.Done() {
					return 0
				}
				return -1
			}
			if bit.Done() {
				return 1
			}
			aa := arena.New(innerType, ait.Next())
			bb := arena.New(innerType, bit.Next())
			if v := compareValues(arena, aa, bb, nullsMax); v != 0 {
				return v
			}
		}
	}
	return bytes.Compare(a.Bytes(), b.Bytes())
}

// SortStable sorts vals according to c, with equal values in their original
// order.  SortStable allocates more memory than [SortStableReader].
func (c *Comparator) SortStable(vals []zed.Value) {
	tmp := make([]zed.Value, len(vals))
	for i, index := range c.sortStableIndices(vals) {
		tmp[i] = vals[i]
		if j := int(index); i < j {
			vals[i] = vals[j]
		} else if i > j {
			vals[i] = tmp[j]
		}
	}
}

// SortStableReader returns a reader for vals sorted according to c, with equal
// values in their original order.
func (c *Comparator) SortStableReader(vals []zed.Value) zio.Reader {
	return &sortStableReader{
		indices: c.sortStableIndices(vals),
		vals:    vals,
	}
}

type sortStableReader struct {
	indices []uint32
	vals    []zed.Value
}

func (s *sortStableReader) Read() (*zed.Value, error) {
	if len(s.indices) == 0 {
		return nil, nil
	}
	val := &s.vals[s.indices[0]]
	s.indices = s.indices[1:]
	return val, nil
}

// SortStable performs a stable sort on the provided records.
func SortStable(records []zed.Value, compare CompareFn) {
	slice := &RecordSlice{records, compare}
	sort.Stable(slice)
}

type RecordSlice struct {
	vals    []zed.Value
	compare CompareFn
}

func NewRecordSlice(compare CompareFn) *RecordSlice {
	return &RecordSlice{compare: compare}
}

// Swap implements sort.Interface for *Record slices.
func (r *RecordSlice) Len() int { return len(r.vals) }

// Swap implements sort.Interface for *Record slices.
func (r *RecordSlice) Swap(i, j int) { r.vals[i], r.vals[j] = r.vals[j], r.vals[i] }

// Less implements sort.Interface for *Record slices.
func (r *RecordSlice) Less(i, j int) bool {
	return r.compare(r.vals[i], r.vals[j]) < 0
}

// Push adds x as element Len(). Implements heap.Interface.
func (r *RecordSlice) Push(rec interface{}) {
	r.vals = append(r.vals, rec.(zed.Value))
}

// Pop removes the first element in the array. Implements heap.Interface.
func (r *RecordSlice) Pop() interface{} {
	rec := r.vals[len(r.vals)-1]
	r.vals = r.vals[:len(r.vals)-1]
	return rec
}

// Index returns the ith record.
func (r *RecordSlice) Index(i int) zed.Value {
	return r.vals[i]
}
