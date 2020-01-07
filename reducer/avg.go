package reducer

import (
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type AvgProto struct {
	target string
	field  string
}

func (ap *AvgProto) Target() string {
	return ap.target
}

func (ap *AvgProto) Instantiate() Interface {
	return &Avg{Field: ap.field}
}

func NewAvgProto(target, field string) *AvgProto {
	return &AvgProto{target, field}
}

type Avg struct {
	Reducer
	Field string
	sum   float64
	count uint64
}

func (a *Avg) Consume(r *zbuf.Record) {
	v := r.ValueByField(a.Field)
	if v == nil {
		a.FieldNotFound++
		return
	}
	var d zng.Double
	if !zng.CoerceToDouble(v, &d) {
		a.TypeMismatch++
		return
	}
	a.sum += float64(d)
	a.count++
}

func (a *Avg) Result() zng.Value {
	var v float64
	if a.count > 0 {
		v = a.sum / float64(a.count)
	}
	return zng.NewDouble(v)
}
