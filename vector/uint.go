package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Uint struct {
	Values []uint64
}

var _ Any = (*Uint)(nil)

func (vector *Uint) newBuilder() builder {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeUint(vector.Values[index]))
		index += 1
	}
}
