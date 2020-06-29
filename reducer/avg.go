package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zngnative"
)

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
	d, ok := zngnative.CoerceToFloat64(v)
	if !ok {
		a.TypeMismatch++
		return
	}
	a.sum += float64(d)
	a.count++
}

func (a *Avg) Result() zng.Value {
	if a.count > 0 {
		return zng.NewFloat64(a.sum / float64(a.count))
	}
	return zng.Value{Type: zng.TypeFloat64}
}

const (
	sumName   = "sum"
	countName = "count"
)

func (a *Avg) ConsumePart(p zng.Value) error {
	rType, ok := p.Type.(*zng.TypeRecord)
	if !ok {
		return ErrBadValue
	}
	rec, err := zng.NewRecord(rType, p.Bytes)
	if err != nil {
		return ErrBadValue
	}
	sumVal, err := rec.ValueByField(sumName)
	if err != nil || sumVal.Type != zng.TypeFloat64 {
		return ErrBadValue
	}
	sum, err := zng.DecodeFloat64(sumVal.Bytes)
	if err != nil {
		return ErrBadValue
	}
	countVal, err := rec.ValueByField(countName)
	if err != nil || countVal.Type != zng.TypeUint64 {
		return ErrBadValue
	}
	count, err := zng.DecodeUint(countVal.Bytes)
	if err != nil {
		return ErrBadValue
	}
	a.sum, a.count = sum, count
	return nil
}

func (a *Avg) ResultPart(zctx *resolver.Context) (zng.Value, error) {
	var zv zcode.Bytes
	zv = zng.NewFloat64(a.sum).Encode(zv)
	zv = zng.NewUint64(a.count).Encode(zv)

	cols := []zng.Column{
		zng.NewColumn(sumName, zng.TypeFloat64),
		zng.NewColumn(countName, zng.TypeUint64),
	}
	typ, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{Type: typ, Bytes: zv}, nil
}
