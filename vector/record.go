package vector

import (
	"github.com/brimdata/zed/zcode"
)

type Record struct {
	Fields []Any
}

var _ Any = (*Record)(nil)

func (vector *Record) newBuilder() builder {
	fieldBuilders := make([]builder, len(vector.Fields))
	for i, field := range vector.Fields {
		fieldBuilders[i] = field.newBuilder()
	}
	return func(builder *zcode.Builder) {
		builder.BeginContainer()
		for _, fieldBuilder := range fieldBuilders {
			fieldBuilder(builder)
		}
		builder.EndContainer()
	}
}
