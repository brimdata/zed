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
	ectx    Context
	fields  []zed.Field
	val     zed.Value
	zctx    *zed.Context
}

func NewSpanFilter(eval Evaluator) *SpanFilter {
	return &SpanFilter{
		eval: eval,
		ectx: NewContext(),
		fields: []zed.Field{
			{Name: "lower"},
			{Name: "upper"},
		},
		zctx: zed.NewContext(),
	}
}

func (o *SpanFilter) Eval(lower, upper *zed.Value) bool {
	o.fields[0].Type = lower.Type
	o.fields[1].Type = upper.Type
	if o.val.Type == nil || o.fields[0].Type != lower.Type || o.fields[1].Type != upper.Type {
		o.val.Type = o.zctx.MustLookupTypeRecord(o.fields)
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
