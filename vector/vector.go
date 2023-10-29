package vector

import (
	"github.com/brimdata/zed"
)

// len(values) == len(Types)
type Vector struct {
	Types  []zed.Type
	values []any
	tags   []int64
}
