package agg

import (
	"github.com/brimdata/zed"
)

type Any struct {
	arena *zed.Arena
	val   *zed.Value
}

var _ Function = (*Any)(nil)

func (a *Any) Consume(val zed.Value) {
	// Copy any value from the input while favoring any-typed non-null values
	// over null values.
	if a.val == nil || a.val.IsNull() && !val.IsNull() {
		if arena, ok := val.Arena(); ok {
			arena.Ref()
			a.arena = arena
		}
		a.val = &val
	}
}

func (a *Any) Result(*zed.Context, *zed.Arena) zed.Value {
	if a.val == nil {
		return zed.Null
	}
	return *a.val
}

func (a *Any) ConsumeAsPartial(_ *zed.Arena, v zed.Value) {
	a.Consume(v)
}

func (a *Any) ResultAsPartial(*zed.Context, *zed.Arena) zed.Value {
	return a.Result(nil, nil)
}
