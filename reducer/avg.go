package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Avg struct {
	Reducer
	Field string
	sum   float64
	count uint64
}

func NewAvg(name, field string) *Avg {
	return &Avg{
		Reducer: New(name),
		Field:   field,
	}
}

func (a *Avg) Consume(r *zson.Record) {
	k, ok := r.ColumnOfField(a.Field)
	if !ok {
		return
	}
	v, err := zeek.UnsafeParseFloat64(r.Slice(k))
	if err != nil {
		a.TypeMismatch++
		return
	}
	a.sum += v
	a.count++
}

func (a *Avg) Result() zeek.Value {
	var v float64
	if a.count > 0 {
		v = a.sum / float64(a.count)
	}
	return &zeek.Double{v}
}
