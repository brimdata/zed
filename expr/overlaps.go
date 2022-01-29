package expr

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#overlaps
type OverlapsFn struct {
	//XXX need to refactor this as something else either proc of function
	zctx *zed.Context
	expr Evaluator
}

func NewOverlaps(zctx *zed.Context, exprs []Evaluator) *OverlapsFn {
	return &OverlapsFn{zctx, exprs[0]}
}

type segment struct {
	Generic
	record *zed.Value
}

//func (o *OverlapsFn) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
func (o *OverlapsFn) Eval(ectx Context, this *zed.Value) *zed.Value {
	arg := o.expr.Eval(ectx, this)
	// array of record with first/last keys
	arrayType, ok := zed.TypeUnder(arg.Type).(*zed.TypeArray)
	if !ok {
		return o.zctx.NewErrorf("overlaps: %q not an array of first/last records", zson.String(arg))
	}
	recType := zed.TypeRecordOf(arrayType.Type)
	if recType == nil {
		return o.zctx.NewErrorf("overlaps: %q not an array of first/last records", zson.String(arg))
	}
	ord := order.Desc //XXX parameter
	cmp := CompareFunc(ord)
	var segments []segment
	it := arg.Iter()
	for !it.Done() {
		val := zed.NewValue(recType, it.Next())
		first := val.Deref("first")
		if first.IsMissing() {
			return o.zctx.NewErrorf("overlaps: item missing 'first' field: %s", zson.String(val))
		}
		last := val.Deref("last")
		if last.IsMissing() {
			fmt.Println(last)
			return o.zctx.NewErrorf("overlaps: item missing 'last' field: %s", zson.String(val))
		}
		segments = append(segments, segment{
			Generic: Generic{
				first: *first,
				last:  *last,
				cmp:   cmp,
			},
			record: zed.NewValue(recType, it.Next()),
		})
	}
	if len(segments) == 0 {
		return zed.Null
	}
	sort.Slice(segments, func(i, j int) bool {
		return segmentLess(segments[i], segments[j])
	})
	keyType := segments[0].first.Type
	itemType := o.zctx.MustLookupTypeRecord([]zed.Column{
		{"first", keyType},
		{"last", keyType},
		{"count", zed.TypeInt64},
	})
	var b zcode.Builder
	var window segset
	// O(n^2) in number of segments
	for _, seg := range segments[1:] {
		window.purge(&seg.first)
		window.add(seg)
		c := window.cover()
		b.BeginContainer()
		b.Append(c.first.Bytes)
		b.Append(c.last.Bytes)
		b.Append(zed.EncodeInt(int64(len(window))))
		b.EndContainer()
	}
	return zed.NewValue(o.zctx.LookupTypeArray(itemType), b.Bytes())
}

type segset []segment

func (s segset) cover() Generic {
	c := s[0].Generic
	for _, seg := range s[1:] {
		c.Extend(&seg.Generic.first)
		c.Extend(&seg.Generic.last)
	}
	return c
}

func (s *segset) add(seg segment) {
	*s = append(*s, seg)
}

func (s *segset) purge(cutoff *zed.Value) {
	var off int
	for _, seg := range *s {
		if seg.After(cutoff) {
			continue
		}
		(*s)[off] = seg
		off++
	}
	*s = (*s)[:off]
	if off > 0 {
		fmt.Println("PURGE", off)
	}
}

func segmentLess(a, b segment) bool {
	if b.Before(a.First()) {
		return true
	}
	if !bytes.Equal(a.First().Bytes, b.First().Bytes) {
		return false
	}
	return a.After(b.Last())
}

//XXX copy extent for now because import cycle

// For now, we do slow-path stuff here but the interface will allow us
// to optimize with type-specific implementations.  It would be trivial here
// to create a Time range that embeds nano.Span instead of Lower/Upper
// and implements Range.

// Span represents the closed interval [first, last] where first is "less than"
// last with respect to the Span's order.Which.
type Span interface {
	First() *zed.Value
	Last() *zed.Value
	Before(*zed.Value) bool
	After(*zed.Value) bool
	In(*zed.Value) bool
	Overlaps(*zed.Value, *zed.Value) bool
	Crop(Span) bool
	Extend(*zed.Value)
	String() string
}

type Generic struct {
	first zed.Value
	last  zed.Value
	cmp   CompareFn
}

// CompareFunc returns a generic comparator suitable for use in a Range
// based on the order of values in the range, i.e., when order is desc
// then the first value is larger than the last value and Before is true
// for larger values while After is true for smaller values, etc.
func CompareFunc(o order.Which) CompareFn {
	// The values of nullsMax here (used during lake data reads) and in
	// zbuf.NewCompareFn (used during lake data writes) must agree.
	cmp := NewValueCompareFn(o == order.Asc)
	if o == order.Asc {
		return cmp
	}
	return func(a, b *zed.Value) int { return cmp(b, a) }
}

// Create a new Range from generic range of zed.Values according
// to lower and upper.  The range is not sensitive to the absolute order
// of lower and upper.
func NewGeneric(lower, upper zed.Value, cmp CompareFn) *Generic {
	if cmp(&lower, &upper) > 0 {
		lower, upper = upper, lower
	}
	return &Generic{
		first: lower,
		last:  upper,
		cmp:   cmp,
	}
}

func NewGenericFromOrder(first, last zed.Value, o order.Which) *Generic {
	return NewGeneric(first, last, CompareFunc(o))
}

func (g *Generic) In(val *zed.Value) bool {
	return g.cmp(val, &g.first) >= 0 && g.cmp(val, &g.last) <= 0
}

func (g *Generic) First() *zed.Value {
	return &g.first
}

func (g *Generic) Last() *zed.Value {
	return &g.last
}

func (g *Generic) After(val *zed.Value) bool {
	return g.cmp(val, &g.last) > 0
}

func (g *Generic) Before(val *zed.Value) bool {
	return g.cmp(val, &g.first) < 0
}

func (g *Generic) Overlaps(first, last *zed.Value) bool {
	if g.cmp(first, &g.first) >= 0 {
		return g.cmp(first, &g.last) <= 0
	}
	return g.cmp(last, &g.first) >= 0
}

func (g *Generic) Crop(s Span) bool {
	if first := s.First(); g.cmp(first, &g.first) > 0 {
		g.first = *first
	}
	if last := s.Last(); g.cmp(last, &g.last) < 0 {
		g.last = *last
	}
	return g.cmp(&g.first, &g.last) <= 0
}

func (g *Generic) Extend(val *zed.Value) {
	if g.cmp(val, &g.first) < 0 {
		g.first = *val.Copy()
	} else if g.cmp(val, &g.last) > 0 {
		g.last = *val.Copy()
	}
}

func (g *Generic) String() string {
	return Format(g)
}

func Format(s Span) string {
	first, err := zson.FormatValue(*s.First())
	if err != nil {
		first = fmt.Sprintf("<%s>", err)
	}
	last, err := zson.FormatValue(*s.Last())
	if err != nil {
		last = fmt.Sprintf("<%s>", err)
	}
	return fmt.Sprintf("first %s last %s", first, last)
}

func Overlaps(a, b Span) bool {
	return !b.Before(a.Last()) && !b.After(a.First())
}
