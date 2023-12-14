package vector

import (
	"github.com/brimdata/zed"
)

// XXX this isn't right
type Vector struct {
	Context *zed.Context
	Types   []zed.Type
	// len(values) == len(Types)
	values []vector
	tags   []int64
}
