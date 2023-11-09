package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

type Materializer struct {
	vector        *Vector
	materializers []materializer
	index         int
	builder       zcode.Builder
	value         zed.Value
}

func (v *Vector) NewMaterializer() Materializer {
	materializers := make([]materializer, len(v.Types))
	for i, value := range v.values {
		materializers[i] = value.newMaterializer()
	}
	return Materializer{
		vector:        v,
		materializers: materializers,
	}
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
	m.index++
	return &m.value, nil
}

type materializer func(*zcode.Builder)

func (v *bools) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeBool(v.values[index]))
		index++
	}
}

func (v *byteses) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		data := v.data[v.offsets[index]:v.offsets[index+1]]
		builder.Append(zed.EncodeBytes(data))
		index++
	}
}

func (v *durations) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeDuration(v.values[index]))
		index++
	}
}

func (v *float16s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeFloat16(v.values[index]))
		index++
	}
}

func (v *float32s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeFloat32(v.values[index]))
		index++
	}
}

func (v *float64s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeFloat64(v.values[index]))
		index++
	}
}

func (v *int8s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeInt(int64(v.values[index])))
		index++
	}
}

func (v *int16s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeInt(int64(v.values[index])))
		index++
	}
}

func (v *int32s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeInt(int64(v.values[index])))
		index++
	}
}

func (v *int64s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeInt(int64(v.values[index])))
		index++
	}
}

func (v *ips) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeIP(v.values[index]))
		index++
	}
}

func (v *nets) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeNet(v.values[index]))
		index++
	}
}

func (v *strings) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		data := v.data[v.offsets[index]:v.offsets[index+1]]
		builder.Append(zed.EncodeBytes(data))
		index++
	}
}

func (v *types) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeTypeValue(v.values[index]))
		index++
	}
}

func (v *times) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeTime(v.values[index]))
		index++
	}
}

func (v *uint8s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeUint(uint64(v.values[index])))
		index++
	}
}

func (v *uint16s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeUint(uint64(v.values[index])))
		index++
	}
}

func (v *uint32s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeUint(uint64(v.values[index])))
		index++
	}
}

func (v *uint64s) newMaterializer() materializer {
	var index int
	return func(builder *zcode.Builder) {
		builder.Append(zed.EncodeUint(uint64(v.values[index])))
		index++
	}
}

func (v *arrays) newMaterializer() materializer {
	var index int
	elemMaterializer := v.elems.newMaterializer()
	return func(builder *zcode.Builder) {
		length := int(v.lengths[index])
		builder.BeginContainer()
		for i := 0; i < length; i++ {
			elemMaterializer(builder)
		}
		builder.EndContainer()
		index++
	}
}

func (v *constants) newMaterializer() materializer {
	return func(builder *zcode.Builder) {
		builder.Append(v.bytes)
	}
}

func (v *maps) newMaterializer() materializer {
	var index int
	keyMaterializer := v.keys.newMaterializer()
	valueMaterializer := v.values.newMaterializer()
	return func(builder *zcode.Builder) {
		length := int(v.lengths[index])
		builder.BeginContainer()
		for i := 0; i < length; i++ {
			keyMaterializer(builder)
			valueMaterializer(builder)
		}
		builder.TransformContainer(zed.NormalizeMap)
		builder.EndContainer()
		index++
	}
}

func (v *nulls) newMaterializer() materializer {
	var runIndex int
	var run int64
	isNull := true
	valueMaterializer := v.values.newMaterializer()
	return func(builder *zcode.Builder) {
		for run == 0 {
			isNull = !isNull
			run = v.runs[runIndex]
			runIndex++
		}
		if isNull {
			builder.Append(nil)
		} else {
			valueMaterializer(builder)
		}
		run--
	}
}

func (v *records) newMaterializer() materializer {
	fieldMaterializers := make([]materializer, len(v.fields))
	for i, field := range v.fields {
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

func (v *unions) newMaterializer() materializer {
	var index int
	payloadMaterializers := make([]materializer, len(v.payloads))
	for i, payload := range v.payloads {
		payloadMaterializers[i] = payload.newMaterializer()
	}
	return func(builder *zcode.Builder) {
		builder.BeginContainer()
		tag := v.tags[index]
		builder.Append(zed.EncodeInt(tag))
		payloadMaterializers[tag](builder)
		builder.EndContainer()
		index++
	}
}
