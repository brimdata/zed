package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// len(values) == len(Types)
type Vector struct {
	Types  []zed.Type
	values []any
	tags   []int64
}

// Materialize a `Value`.
// Returns false if no more values exist.
type Materializer func() (*zed.Value, bool)

func (vector *Vector) NewMaterializer() Materializer {
	var index int
	var builder zcode.Builder
	types := vector.Types
	length := len(vector.tags)
	tags := vector.tags
	materializers := make([]materializer, len(vector.Types))
	for i, value := range vector.values {
		materializers[i] = value.newMaterializer()
	}
	return func() (*zed.Value, bool) {
		if index >= length {
			return nil, false
		}
		tag := tags[index]
		typ := types[tag]
		builder.Truncate()
		materializers[tag](&builder)
		value := *zed.NewValue(typ, builder.Bytes().Body())
		return &value, true
	}
}

// TODO This exists as a builtin in go 1.21
func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
