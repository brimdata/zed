package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type ObjectFilter struct {
	eval Evaluator

	builder zcode.Builder
	ectx    Context
	cols    []zed.Column
	typ     zed.Type
	zctx    *zed.Context
}

func NewObjectFilter(eval Evaluator) *ObjectFilter {
	return &ObjectFilter{
		eval: eval,
		ectx: NewContext(),
		cols: []zed.Column{
			{Name: "lower"},
			{Name: "upper"},
		},
		zctx: zed.NewContext(),
	}
}

func (o *ObjectFilter) Eval(lower, upper *zed.Value) bool {
	o.cols[0].Type = lower.Type
	o.cols[1].Type = upper.Type
	if o.typ == nil || o.cols[0].Type != lower.Type && o.cols[1].Type != upper.Type {
		o.typ = o.zctx.MustLookupTypeRecord(o.cols)
	}
	o.builder.Reset()
	o.builder.Append(lower.Bytes)
	o.builder.Append(upper.Bytes)
	val := o.eval.Eval(o.ectx, o.ectx.NewValue(o.typ, o.builder.Bytes()))
	return !zed.DecodeBool(val.Bytes)
}
