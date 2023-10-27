package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type String struct {
	Values []string
}

var _ Any = (*String)(nil)

func (vector *String) newBuilder() builder {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeString(vector.Values[index]))
		index += 1
	}
}
