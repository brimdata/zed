package reducer

import (
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zx"
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
	v, err := r.ValueByField(a.Field)
	if err != nil {
		a.FieldNotFound++
		return
	}
	d, ok := zx.CoerceToDouble(v)
	if !ok {
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
