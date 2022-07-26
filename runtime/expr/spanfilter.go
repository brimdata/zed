package expr

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// SpanFilter is a filter for a span or range of values.
type SpanFilter struct {
	eval Evaluator

	builder zcode.Builder
	cols    []zed.Column
	ectx    Context
	val     zed.Value
	zctx    *zed.Context
}

func NewSpanFilter(eval Evaluator) *SpanFilter {
	return &SpanFilter{
		eval: eval,
		cols: []zed.Column{
			{Name: "lower"},
			{Name: "upper"},
		},
		ectx: NewContext(),
		zctx: zed.NewContext(),
	}
}

func (o *SpanFilter) Eval(lower, upper *zed.Value) bool {
	o.cols[0].Type = lower.Type
	o.cols[1].Type = upper.Type
	if o.val.Type == nil || o.cols[0].Type != lower.Type && o.cols[1].Type != upper.Type {
		o.val.Type = o.zctx.MustLookupTypeRecord(o.cols)
	}
	o.builder.Reset()
	o.builder.Append(lower.Bytes)
	o.builder.Append(upper.Bytes)
	o.val.Bytes = o.builder.Bytes()
	val := o.eval.Eval(o.ectx, &o.val)
	if val.Type != zed.TypeBool {
		panic(fmt.Errorf("result of SpanFilter not a boolean: %s", zson.FormatType(val.Type)))
	}
	return !zed.DecodeBool(val.Bytes)
}
