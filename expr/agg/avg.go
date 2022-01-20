package agg

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Avg struct {
	sum   float64
	count uint64
}

var _ Function = (*Avg)(nil)

func (a *Avg) Consume(val *zed.Value) {
	if val.IsNull() {
		return
	}
	if d, ok := coerce.ToFloat(*val); ok {
		a.sum += float64(d)
		a.count++
	}
}

func (a *Avg) Result(*zed.Context) *zed.Value {
	if a.count > 0 {
		avg := a.sum / float64(a.count)
		return zed.NewValue(zed.TypeFloat64, zed.EncodeFloat64(avg))
	}
	return zed.NullFloat64
}

const (
	sumName   = "sum"
	countName = "count"
)

func (a *Avg) ConsumeAsPartial(partial *zed.Value) {
	recType := zed.TypeRecordOf(partial.Type)
	if recType == nil {
		panic(fmt.Errorf("avg: partial is not a record: %s", zson.MustFormatValue(*partial)))
	}
	rec := zed.NewValue(recType, partial.Bytes)
	sumVal, err := rec.ValueByField(sumName)
	if err != nil {
		panic(fmt.Errorf("avg: partial sum is missing: %w", err))
	}
	if sumVal.Type != zed.TypeFloat64 {
		panic(fmt.Errorf("avg: partial sum has bad type: %s", zson.MustFormatValue(sumVal)))
	}
	countVal, err := rec.ValueByField(countName)
	if err != nil {
		panic(err)
	}
	if countVal.Type != zed.TypeUint64 {
		panic(fmt.Errorf("avg: partial count has bad type: %s", zson.MustFormatValue(countVal)))
	}
	a.sum += zed.DecodeFloat64(sumVal.Bytes)
	a.count += zed.DecodeUint(countVal.Bytes)
}

func (a *Avg) ResultAsPartial(zctx *zed.Context) *zed.Value {
	var zv zcode.Bytes
	zv = zed.NewFloat64(a.sum).Encode(zv)
	zv = zed.NewUint64(a.count).Encode(zv)
	cols := []zed.Column{
		zed.NewColumn(sumName, zed.TypeFloat64),
		zed.NewColumn(countName, zed.TypeUint64),
	}
	typ := zctx.MustLookupTypeRecord(cols)
	return zed.NewValue(typ, zv)
}
