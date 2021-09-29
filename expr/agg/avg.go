package agg

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zcode"
)

type Avg struct {
	sum   float64
	count uint64
}

func (a *Avg) Consume(v zed.Value) error {
	if v.Bytes == nil {
		return nil
	}
	if d, ok := coerce.ToFloat(v); ok {
		a.sum += float64(d)
		a.count++
	}
	return nil
}

func (a *Avg) Result(*zed.Context) (zed.Value, error) {
	if a.count > 0 {
		return zed.NewFloat64(a.sum / float64(a.count)), nil
	}
	return zed.Value{Type: zed.TypeFloat64}, nil
}

const (
	sumName   = "sum"
	countName = "count"
)

func (a *Avg) ConsumeAsPartial(p zed.Value) error {
	rType, ok := p.Type.(*zed.TypeRecord)
	if !ok {
		return ErrBadValue
	}
	rec := zed.NewRecord(rType, p.Bytes)
	sumVal, err := rec.ValueByField(sumName)
	if err != nil || sumVal.Type != zed.TypeFloat64 {
		return ErrBadValue
	}
	sum, err := zed.DecodeFloat64(sumVal.Bytes)
	if err != nil {
		return ErrBadValue
	}
	countVal, err := rec.ValueByField(countName)
	if err != nil || countVal.Type != zed.TypeUint64 {
		return ErrBadValue
	}
	count, err := zed.DecodeUint(countVal.Bytes)
	if err != nil {
		return ErrBadValue
	}
	a.sum += sum
	a.count += count
	return nil
}

func (a *Avg) ResultAsPartial(zctx *zed.Context) (zed.Value, error) {
	var zv zcode.Bytes
	zv = zed.NewFloat64(a.sum).Encode(zv)
	zv = zed.NewUint64(a.count).Encode(zv)

	cols := []zed.Column{
		zed.NewColumn(sumName, zed.TypeFloat64),
		zed.NewColumn(countName, zed.TypeUint64),
	}
	typ, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return zed.Value{}, err
	}
	return zed.Value{Type: typ, Bytes: zv}, nil
}
