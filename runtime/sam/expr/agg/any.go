package agg

import (
	"github.com/brimdata/super"
)

type Any zed.Value

var _ Function = (*Any)(nil)

func NewAny() *Any {
	a := (Any)(zed.Null)
	return &a
}

func (a *Any) Consume(val zed.Value) {
	// Copy any value from the input while favoring any-typed non-null values
	// over null values.
	if (*zed.Value)(a).Type() == nil || (*zed.Value)(a).IsNull() && !val.IsNull() {
		*a = Any(val.Copy())
	}
}

func (a *Any) Result(*zed.Context) zed.Value {
	if (*zed.Value)(a).Type() == nil {
		return zed.Null
	}
	return *(*zed.Value)(a)
}

func (a *Any) ConsumeAsPartial(v zed.Value) {
	a.Consume(v)
}

func (a *Any) ResultAsPartial(*zed.Context) zed.Value {
	return a.Result(nil)
}
