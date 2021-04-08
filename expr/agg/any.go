package agg

import (
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Any zng.Value

func (a *Any) Consume(v zng.Value) error {
	// Copy any value from the input while favoring any-typed non-null values
	// over null values.
	if a.Type == nil || (a.Bytes == nil && v.Bytes != nil) {
		*a = Any(v.Copy())
	}
	return nil
}

func (a Any) Result(*zson.Context) (zng.Value, error) {
	if a.Type == nil {
		return zng.Value{Type: zng.TypeNull}, nil
	}
	return zng.Value(a), nil
}

func (a *Any) ConsumeAsPartial(v zng.Value) error {
	return a.Consume(v)
}

func (a Any) ResultAsPartial(*zson.Context) (zng.Value, error) {
	return a.Result(nil)
}
