package extent

import (
	"fmt"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

// For now, we do slow-path stuff here but the interface will allow us
// to optimize with type-specific implementations.  It would be trivial here
// to create a Time range that embeds nano.Span instead of Lower/Upper
// and implements Range.

// Span represents the closed interval [first, last] where first is "less than"
// last with respect to the Span's order.Which.
type Span interface {
	First() zng.Value
	Last() zng.Value
	Before(zng.Value) bool
	After(zng.Value) bool
	In(zng.Value) bool
	Overlaps(zng.Value, zng.Value) bool
	Crop(Span) bool
	Extend(zng.Value)
	String() string
}

type Generic struct {
	first zng.Value
	last  zng.Value
	cmp   expr.ValueCompareFn
}

// CompareFunc returns a generic comparator suitable for use in a Range
// based on the order of values in the range, i.e., when order is desc
// then the first value is larger than the last value and Before is true
// for larger values while After is true for smaller values, etc.
func CompareFunc(o order.Which) expr.ValueCompareFn {
	//  Treat null as the minimum value.
	cmp := expr.NewValueCompareFn(false)
	if o == order.Asc {
		return cmp
	}
	return func(a, b zng.Value) int { return cmp(b, a) }
}

// Create a new Range from generic range of zng.Values according
// to lower and upper.  The range is not sensitive to the absolute order
// of lower and upper.
func NewGeneric(lower, upper zng.Value, cmp expr.ValueCompareFn) *Generic {
	if cmp(lower, upper) > 0 {
		lower, upper = upper, lower
	}
	return &Generic{
		first: lower,
		last:  upper,
		cmp:   cmp,
	}
}

func NewGenericFromOrder(first, last zng.Value, o order.Which) *Generic {
	return NewGeneric(first, last, CompareFunc(o))
}

func (g *Generic) In(zv zng.Value) bool {
	return g.cmp(zv, g.first) >= 0 && g.cmp(zv, g.last) <= 0
}

func (g *Generic) First() zng.Value {
	return g.first
}

func (g *Generic) Last() zng.Value {
	return g.last
}

func (g *Generic) After(zv zng.Value) bool {
	return g.cmp(zv, g.last) > 0
}

func (g *Generic) Before(zv zng.Value) bool {
	return g.cmp(zv, g.first) < 0
}

func (g *Generic) Overlaps(first, last zng.Value) bool {
	if g.cmp(first, g.first) >= 0 {
		return g.cmp(first, g.last) <= 0
	}
	return g.cmp(last, g.first) >= 0
}

func (g *Generic) Crop(s Span) bool {
	if first := s.First(); g.cmp(first, g.first) > 0 {
		g.first = first
	}
	if last := s.Last(); g.cmp(last, g.last) < 0 {
		g.last = last
	}
	return g.cmp(g.first, g.last) <= 0
}

func (g *Generic) Extend(zv zng.Value) {
	if g.cmp(zv, g.first) < 0 {
		g.first = zv
	} else if g.cmp(zv, g.last) > 0 {
		g.last = zv
	}
}

func (g *Generic) String() string {
	return Format(g)
}

func Format(s Span) string {
	first, err := zson.FormatValue(s.First())
	if err != nil {
		first = fmt.Sprintf("<%s>", err)
	}
	last, err := zson.FormatValue(s.Last())
	if err != nil {
		last = fmt.Sprintf("<%s>", err)
	}
	return fmt.Sprintf("first %s last %s", first, last)
}

func Overlaps(a, b Span) bool {
	return !b.Before(a.Last()) && !b.After(a.First())
}
