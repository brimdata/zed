package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Int struct {
	Values []int64
}

var _ Any = (*Int)(nil)

func (vector *Int) newBuilder() builder {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeInt(vector.Values[index]))
		index += 1
	}
}
