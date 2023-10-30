package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

func (vector *Vector) NewMaterializer() Materializer {
	materializers := make([]materializer, len(vector.Types))
	for i, value := range vector.values {
		materializers[i] = value.newMaterializer()
	}
	return Materializer{
		vector:        vector,
		materializers: materializers,
	}
}

type Materializer struct {
	vector        *Vector
	materializers []materializer
	index         int
	builder       zcode.Builder
	value         zed.Value
}

var _ zio.Reader = (*Materializer)(nil)

func (m *Materializer) Read() (*zed.Value, error) {
	if m.index >= len(m.vector.tags) {
		return nil, nil
	}
	tag := m.vector.tags[m.index]
	typ := m.vector.Types[tag]
	m.builder.Truncate()
	m.materializers[tag](&m.builder)
	m.value = *zed.NewValue(typ, m.builder.Bytes().Body())
	m.index += 1
	return &m.value, nil
}

// TODO This exists as a builtin in go 1.21
func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

type materializer func(*zcode.Builder)

func (vector *bools) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeBool(vector.values[index]))
		index += 1
	}
}

func (vector *byteses) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeBytes(vector.values[index]))
		index += 1
	}
}

func (vector *durations) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeDuration(vector.values[index]))
		index += 1
	}
}

func (vector *float16s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeFloat16(vector.values[index]))
		index += 1
	}
}

func (vector *float32s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeFloat32(vector.values[index]))
		index += 1
	}
}

func (vector *float64s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeFloat64(vector.values[index]))
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

func (vector *ips) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeIP(vector.values[index]))
		index += 1
	}
}

func (vector *nets) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeNet(vector.values[index]))
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

func (vector *times) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeTime(vector.values[index]))
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
		if vector.mask.ContainsInt(index) {
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
