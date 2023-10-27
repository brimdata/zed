package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Map struct {
	Lengths []uint32
	Keys    Any
	Values  Any
}

var _ Any = (*Map)(nil)

func (vector *Map) newBuilder() builder {
	var index int
	keyBuilder := vector.Keys.newBuilder()
	valueBuilder := vector.Values.newBuilder()
	return func(builder *zcode.Builder) {
		length := int(vector.Lengths[index])
		builder.BeginContainer()
		for i := 0; i < length; i += 1 {
			keyBuilder(builder)
			valueBuilder(builder)
		}
		builder.TransformContainer(zed.NormalizeMap)
		builder.EndContainer()
		index += 1
	}
}
