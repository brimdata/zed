package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Union struct {
	Tags     []uint32
	Payloads []Any
}

var _ Any = (*Union)(nil)

func (vector *Union) newBuilder() builder {
	var index int
	payloadBuilders := make([]builder, len(vector.Payloads))
	for i, payload := range vector.Payloads {
		payloadBuilders[i] = payload.newBuilder()
	}
	return func(builder *zcode.Builder) {
		builder.BeginContainer()
		tag := vector.Tags[index]
		builder.Append(zed.EncodeInt(int64(tag)))
		payloadBuilders[tag](builder)
		builder.EndContainer()
		index += 1
	}
}
