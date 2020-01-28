package reducer

import (
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

func (ap *AvgProto) Instantiate(*zng.TypeRecord) Interface {
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

func (a *Avg) Consume(r *zng.Record) {
	v, err := r.ValueByField(a.Field)
	if err != nil {
		a.FieldNotFound++
		return
	}
	if v.Bytes == nil {
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
	if a.count > 0 {
		return zng.NewDouble(a.sum / float64(a.count))
	}
	return zng.Value{Type: zng.TypeDouble}
}
