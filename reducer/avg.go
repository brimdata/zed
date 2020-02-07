package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
)

type AvgProto struct {
	target   string
	resolver expr.FieldExprResolver
}

func (ap *AvgProto) Target() string {
	return ap.target
}

func (ap *AvgProto) Instantiate(*zng.Record) Interface {
	return &Avg{Resolver: ap.resolver}
}

func NewAvgProto(target string, field expr.FieldExprResolver) *AvgProto {
	return &AvgProto{target, field}
}

type Avg struct {
	Reducer
	Resolver expr.FieldExprResolver
	sum      float64
	count    uint64
}

func (a *Avg) Consume(r *zng.Record) {
	v := a.Resolver(r)
	if v.Type == nil {
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
