package agg

import (
	"github.com/brimdata/zq/expr/coerce"
	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

type Avg struct {
	sum   float64
	count uint64
}

func (a *Avg) Consume(v zng.Value) error {
	if v.Bytes == nil {
		return nil
	}
	if d, ok := coerce.ToFloat(v); ok {
		a.sum += float64(d)
		a.count++
	}
	return nil
}

func (a *Avg) Result(*resolver.Context) (zng.Value, error) {
	if a.count > 0 {
		return zng.NewFloat64(a.sum / float64(a.count)), nil
	}
	return zng.Value{Type: zng.TypeFloat64}, nil
}

const (
	sumName   = "sum"
	countName = "count"
)

func (a *Avg) ConsumeAsPartial(p zng.Value) error {
	rType, ok := p.Type.(*zng.TypeRecord)
	if !ok {
		return ErrBadValue
	}
	rec := zng.NewRecord(rType, p.Bytes)
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
	a.sum += sum
	a.count += count
	return nil
}

func (a *Avg) ResultAsPartial(zctx *resolver.Context) (zng.Value, error) {
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
