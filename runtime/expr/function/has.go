package function

import "github.com/brimdata/zed"

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#has
type Has struct{}

func (h *Has) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	for _, val := range args {
		if val.IsError() {
			if val.IsMissing() || val.IsQuiet() {
				return zed.False
			}
			return &val
		}
	}
	return zed.True
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#missing
type Missing struct {
	has Has
}

func (m *Missing) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := m.has.Call(ctx, args)
	if val.Type == zed.TypeBool {
		val = zed.Not(val.Bytes)
	}
	return val
}
