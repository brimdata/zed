package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
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

func (a *Avg) Consume(r *zson.Record) {
	v := r.ValueByField(a.Field)
	if v == nil {
		a.FieldNotFound++
		return
	}
	var d zeek.Double
	if !zeek.CoerceToDouble(v, &d) {
		a.TypeMismatch++
		return
	}
	a.sum += float64(d)
	a.count++
}

func (a *Avg) Result() zeek.Value {
	var v float64
	if a.count > 0 {
		v = a.sum / float64(a.count)
	}
	return zeek.NewDouble(v)
}
