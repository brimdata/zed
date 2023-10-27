package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Bool struct {
	Values []bool // TODO Use bitset.
}

var _ Any = (*Bool)(nil)

func (vector *Bool) newBuilder() builder {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeBool(vector.Values[index]))
		index += 1
	}
}
