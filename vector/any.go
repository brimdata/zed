package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Any interface {
	Type() zed.Type
	Len() uint32
	Serialize(*zcode.Builder, uint32)
}

type Puller interface {
	Pull(done bool) (Any, error)
}

type Builder func(*zcode.Builder) bool
