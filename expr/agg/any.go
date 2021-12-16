package agg

import (
	"github.com/brimdata/zed"
)

type Any zed.Value

var _ Function = (*Any)(nil)

func NewAny() *Any {
	return (*Any)(zed.NewValue(zed.TypeNull, nil))
}

func (a *Any) Consume(val *zed.Value) {
	// Copy any value from the input while favoring any-typed non-null values
	// over null values.
	if a.Type == nil || (a.Bytes == nil && val.Bytes != nil) {
		*a = Any(*val.Copy())
	}
}

func (a *Any) Result(*zed.Context) *zed.Value {
	if a.Type == nil {
		return zed.Null
	}
	return (*zed.Value)(a)
}

func (a *Any) ConsumeAsPartial(v *zed.Value) {
	a.Consume(v)
}

func (a Any) ResultAsPartial(*zed.Context) *zed.Value {
	return a.Result(nil)
}
