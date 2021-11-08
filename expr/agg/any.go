package agg

import (
	"github.com/brimdata/zed"
)

type Any zed.Value

func (a *Any) Consume(v zed.Value) error {
	// Copy any value from the input while favoring any-typed non-null values
	// over null values.
	if a.Type == nil || (a.Bytes == nil && v.Bytes != nil) {
		*a = Any(*v.Copy())
	}
	return nil
}

func (a Any) Result(*zed.Context) (zed.Value, error) {
	if a.Type == nil {
		return zed.Value{Type: zed.TypeNull}, nil
	}
	return zed.Value(a), nil
}

func (a *Any) ConsumeAsPartial(v zed.Value) error {
	return a.Consume(v)
}

func (a Any) ResultAsPartial(*zed.Context) (zed.Value, error) {
	return a.Result(nil)
}
