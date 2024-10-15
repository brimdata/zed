package function

import "github.com/brimdata/super"

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#has
type Has struct{}

func (h *Has) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	for _, val := range args {
		if val.IsError() {
			if val.IsMissing() || val.IsQuiet() {
				return zed.False
			}
			return val
		}
	}
	return zed.True
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#missing
type Missing struct {
	has Has
}

func (m *Missing) Call(ectx zed.Allocator, args []zed.Value) zed.Value {
	val := m.has.Call(ectx, args)
	if val.Type() == zed.TypeBool {
		return zed.NewBool(!val.Bool())
	}
	return val
}
