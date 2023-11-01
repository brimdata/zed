package vector

import (
	"github.com/brimdata/zed"
)

type Vector struct {
	Context *zed.Context
	Types   []zed.Type
	// len(values) == len(Types)
	values []vector
	tags   []int64
}
