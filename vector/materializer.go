package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type materializer func(*zcode.Builder)

func (vector *bools) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeBool(vector.values[index]))
		index += 1
	}
}

func (vector *ints) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeInt(vector.values[index]))
		index += 1
	}
}

func (vector *strings) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeString(vector.values[index]))
		index += 1
	}
}

func (vector *uints) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeUint(vector.values[index]))
		index += 1
	}
}

func (vector *arrays) newMaterializer() materializer {
	var index int
	elemMaterializer := vector.elems.newMaterializer()
	return func(builder *zcode.Builder) {
		length := int(vector.lengths[index])
		builder.BeginContainer()
		for i := 0; i < length; i += 1 {
			elemMaterializer(builder)
		}
		builder.EndContainer()
		index += 1
	}
}

func (vector *constants) newMaterializer() materializer {
	bytes := vector.value.Bytes()
	return func(builder *zcode.Builder) {
		builder.Append(bytes)
	}
}

func (vector *maps) newMaterializer() materializer {
	var index int
	keyMaterializer := vector.keys.newMaterializer()
	valueMaterializer := vector.values.newMaterializer()
	return func(builder *zcode.Builder) {
		length := int(vector.lengths[index])
		builder.BeginContainer()
		for i := 0; i < length; i += 1 {
			keyMaterializer(builder)
			valueMaterializer(builder)
		}
		builder.TransformContainer(zed.NormalizeMap)
		builder.EndContainer()
		index += 1
	}
}

func (vector *nulls) newMaterializer() materializer {
	var index int
	valueMaterializer := vector.values.newMaterializer()
	return func(builder *zcode.Builder) {
		if vector.Has(int64(index)) {
			valueMaterializer(builder)
		} else {
			builder.Append(nil)
		}
		index += 1
	}
}

func (vector *records) newMaterializer() materializer {
	fieldMaterializers := make([]materializer, len(vector.fields))
	for i, field := range vector.fields {
		fieldMaterializers[i] = field.newMaterializer()
	}
	return func(builder *zcode.Builder) {
		builder.BeginContainer()
		for _, fieldMaterializer := range fieldMaterializers {
			fieldMaterializer(builder)
		}
		builder.EndContainer()
	}
}

func (vector *unions) newMaterializer() materializer {
	var index int
	payloadMaterializers := make([]materializer, len(vector.payloads))
	for i, payload := range vector.payloads {
		payloadMaterializers[i] = payload.newMaterializer()
	}
	return func(builder *zcode.Builder) {
		builder.BeginContainer()
		tag := vector.tags[index]
		builder.Append(zed.EncodeInt(tag))
		payloadMaterializers[tag](builder)
		builder.EndContainer()
		index += 1
	}
}
