package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Const struct {
	Value zed.Value
}

var _ Any = (*Const)(nil)

func (vector *Const) newBuilder() builder {
	bytes := vector.Value.Bytes()
	return func(builder *zcode.Builder) {
		builder.Append(bytes)
	}
}
