package vector

import (
	"github.com/brimdata/zed/zcode"
)

type Array struct {
	Lengths []uint32
	Elems   Any
}

var _ Any = (*Array)(nil)

func (vector *Array) newBuilder() builder {
	var index int
	elemBuilder := vector.Elems.newBuilder()
	return func(builder *zcode.Builder) {
		length := int(vector.Lengths[index])
		builder.BeginContainer()
		for i := 0; i < length; i += 1 {
			elemBuilder(builder)
		}
		builder.EndContainer()
		index += 1
	}
}
