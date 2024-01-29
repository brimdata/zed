package agg

import (
	"github.com/brimdata/zed"
)

type Any struct {
	arena *zed.Arena
	val   zed.Value
}

var _ Function = (*Any)(nil)

func NewAny() *Any {
	return &Any{val: zed.Null}
}

func (a *Any) Consume(val zed.Value) {
	// Copy any value from the input while favoring any-typed non-null values
	// over null values.
	if a.val.Type() == nil || a.val.IsNull() && !val.IsNull() {
		a.arena, _ = val.Arena()
		a.val = val.CopyToArena(a.arena)
	}
}

func (a *Any) Result(*zed.Arena) zed.Value {
	return a.val
}

func (a *Any) ConsumeAsPartial(v zed.Value) {
	a.Consume(v)
}

func (a *Any) ResultAsPartial(*zed.Arena) zed.Value {
	return a.Result(nil)
}
