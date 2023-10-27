package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Vectors struct {
	Type   zed.Type
	Len    int
	Values Any
}

type Any interface {
	// Returned `builder` panics if called more than `Len` times.
	newBuilder() builder
}

// TODO Take an argument that specifies how many values to build.
type builder func(*zcode.Builder)

// TODO Is Uint32 sufficient for Lengths?
// TODO Is Uint32 sufficient for Tags?
// TODO How to handle (u)ints larger than 64 bits?
