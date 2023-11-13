package vector

import (
	"bytes"
	"fmt"
	"io"
	"net/netip"
	"slices"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/vng"
	vngvector "github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

func Read(reader *vng.Reader, demandOut demand.Demand) (*Vector, error) {
	context := zed.NewContext()
	tags, err := ReadInt64s(reader.Root)
	if err != nil {
		return nil, err
	}
	types := make([]zed.Type, len(reader.Readers))
	values := make([]vector, len(reader.Readers))
	for i, typedReader := range reader.Readers {
		typeCopy, _ := context.DecodeTypeValue(zed.EncodeTypeValue(typedReader.Type))
		typ := typeAfterDemand(context, typedReader.Reader, demandOut, typeCopy)
		types[i] = typ
		value, err := read(context, typedReader.Reader, demandOut)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	vector := &Vector{
		Context: context,
		Types:   types,
		values:  values,
		tags:    tags,
	}
	return vector, nil
}

func read(context *zed.Context, reader vngvector.Reader, demandOut demand.Demand) (vector, error) {
	if demand.IsNone(demandOut) {
		return &constants{}, nil
	}

	switch reader := reader.(type) {

	case *vngvector.ArrayReader:
		lengths, err := ReadInt64s(reader.Lengths)
		if err != nil {
			return nil, err
		}
		elems, err := read(context, reader.Elems, demand.All())
		if err != nil {
			return nil, err
		}
		vector := &arrays{
			lengths: lengths,
			elems:   elems,
		}
		return vector, nil

	case *vngvector.ConstReader:
		var builder zcode.Builder
		err := reader.Read(&builder)
		if err != nil {
			return nil, err
		}
		it := zcode.Bytes(builder.Bytes()).Iter()
		vector := &constants{
			bytes: it.Next(),
		}
		return vector, nil

	case *vngvector.DictReader:
		panic("TODO")

	case *vngvector.MapReader:
		keys, err := read(context, reader.Keys, demand.All())
		if err != nil {
			return nil, err
		}
		lengths, err := ReadInt64s(reader.Lengths)
		if err != nil {
			return nil, err
		}
		values, err := read(context, reader.Values, demand.All())
		if err != nil {
			return nil, err
		}
		vector := &maps{
			lengths: lengths,
			keys:    keys,
			values:  values,
		}
		return vector, nil

	case *vngvector.NullsReader:
		runs, err := ReadInt64s(&reader.Runs)
		if err != nil {
			return nil, err
		}
		values, err := read(context, reader.Values, demandOut)
		if err != nil {
			return nil, err
		}
		if len(runs) == 0 {
			return values, nil
		}
		vector := &nulls{
			runs:   runs,
			values: values,
		}
		return vector, nil

	case *vngvector.PrimitiveReader:
		return readPrimitive(context, reader.Typ, reader.Segmap(), reader.Reader())

	case *vngvector.RecordReader:
		var fields []vector
		for i, fieldReader := range reader.Values {
			demandValueOut := demand.GetKey(demandOut, reader.Names[i])
			if !demand.IsNone(demandValueOut) {
				field, err := read(context, fieldReader.Values, demandValueOut)
				if err != nil {
					return nil, err
				}
				fields = append(fields, field)
			}
		}
		vector := &records{
			fields: fields,
		}
		return vector, nil

	case *vngvector.UnionReader:
		payloads := make([]vector, len(reader.Readers))
		for i, reader := range reader.Readers {
			payload, err := read(context, reader, demandOut)
			if err != nil {
				return nil, err
			}
			payloads[i] = payload
		}
		tags, err := ReadInt64s(reader.Tags)
		if err != nil {
			return nil, err
		}
		vector := &unions{
			payloads: payloads,
			tags:     tags,
		}
		return vector, nil

	default:
		return nil, fmt.Errorf("unknown VNG vector Reader type: %T", reader)
	}
}

func ReadInt64s(reader *vngvector.Int64Reader) ([]int64, error) {
	segmap := reader.PrimitiveReader.Segmap()
	readerAt := reader.PrimitiveReader.Reader()
	vector, err := readPrimitive(nil, zed.TypeInt64, segmap, readerAt)
	if err != nil {
		return nil, err
	}
	return vector.(*int64s).values, nil
}

// TODO This is likely to be a bottleneck. If so, inline `readBytes` and `zed.Decode*`.
func readPrimitive(context *zed.Context, typ zed.Type, segmap []vngvector.Segment, readerAt io.ReaderAt) (vector, error) {
	var buf []byte
	switch typ {
	case zed.TypeBool:
		var values []bool
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				value := zed.DecodeBool(bs)
				values = append(values, value)
			}
		}
		vector := &bools{
			values: values,
		}
		return vector, nil

	case zed.TypeBytes:
		data := bytes.NewBuffer(nil)
		offsets := []int{0}
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				data.Write(zed.DecodeBytes(bs))
				offsets = append(offsets, data.Len())
			}
		}
		vector := &byteses{
			data: data.Bytes(),
			// TODO truncate offsets
			offsets: offsets,
		}
		return vector, nil

	case zed.TypeDuration:
		var values []nano.Duration
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeDuration(bs))
			}
		}
		vector := &durations{
			values: values,
		}
		return vector, nil

	case zed.TypeFloat16:
		var values []float32
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeFloat16(bs))
			}
		}
		vector := &float16s{
			values: values,
		}
		return vector, nil

	case zed.TypeFloat32:
		var values []float32
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeFloat32(bs))
			}
		}
		vector := &float32s{
			values: values,
		}
		return vector, nil

	case zed.TypeFloat64:
		var values []float64
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeFloat64(bs))
			}
		}
		vector := &float64s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt8:
		var values []int8
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				value := int8(zed.DecodeInt(bs))
				values = append(values, value)
			}
		}
		vector := &int8s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt16:
		var values []int16
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				value := int16(zed.DecodeInt(bs))
				values = append(values, value)
			}
		}
		vector := &int16s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt32:
		var values []int32
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				value := int32(zed.DecodeInt(bs))
				values = append(values, value)
			}
		}
		vector := &int32s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt64:
		var values []int64
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				value := int64(zed.DecodeInt(bs))
				values = append(values, value)
			}
		}
		vector := &int64s{
			values: values,
		}
		return vector, nil

	case zed.TypeIP:
		var values []netip.Addr
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeIP(bs))
			}
		}
		vector := &ips{
			values: values,
		}
		return vector, nil

	case zed.TypeNet:
		var values []netip.Prefix
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeNet(bs))
			}
		}
		vector := &nets{
			values: values,
		}
		return vector, nil

	case zed.TypeString:
		data := bytes.NewBuffer(nil)
		offsets := []int{0}
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				data.Write(zed.DecodeBytes(bs))
				offsets = append(offsets, data.Len())
			}
		}
		vector := &strings{
			data: data.Bytes(),
			// TODO truncate offsets
			offsets: offsets,
		}
		return vector, nil

	case zed.TypeTime:
		var values []nano.Ts
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, zed.DecodeTime(bs))
			}
		}
		vector := &times{
			values: values,
		}
		return vector, nil

	case zed.TypeUint8:
		var values []uint8
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, uint8(zed.DecodeUint(bs)))
			}
		}
		vector := &uint8s{
			values: values,
		}
		return vector, nil

	case zed.TypeUint16:
		var values []uint16
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, uint16(zed.DecodeUint(bs)))
			}
		}
		vector := &uint16s{
			values: values,
		}
		return vector, nil

	case zed.TypeUint32:
		var values []uint32
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, uint32(zed.DecodeUint(bs)))
			}
		}
		vector := &uint32s{
			values: values,
		}
		return vector, nil

	case zed.TypeUint64:
		var values []uint64
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				values = append(values, uint64(zed.DecodeUint(bs)))
			}
		}
		vector := &uint64s{
			values: values,
		}
		return vector, nil

	case zed.TypeNull:
		return &constants{}, nil

	case zed.TypeType:
		var values []zed.Type
		for _, segment := range segmap {
			buf = slices.Grow(buf[:0], int(segment.MemLength))[:segment.MemLength]
			if err := segment.Read(readerAt, buf); err != nil {
				return nil, err
			}
			it := zcode.Iter(buf)
			for !it.Done() {
				bs := it.Next()
				typ, _ := context.DecodeTypeValue(bs)
				values = append(values, typ)
			}
		}
		vector := &types{
			values: values,
		}
		return vector, nil

	default:
		return nil, fmt.Errorf("unknown VNG type: %T", typ)
	}
}

// This must match exactly the effects of demand on `read`.
func typeAfterDemand(context *zed.Context, reader vngvector.Reader, demandOut demand.Demand, typ zed.Type) zed.Type {
	if demand.IsNone(demandOut) {
		return zed.TypeNull
	}
	if demand.IsAll(demandOut) {
		return typ
	}
	switch reader := reader.(type) {
	case *vngvector.NullsReader:
		return typeAfterDemand(context, reader.Values, demandOut, typ)

	case *vngvector.RecordReader:
		typ := typ.(*zed.TypeRecord)
		var fields []zed.Field
		for i, fieldReader := range reader.Values {
			demandValueOut := demand.GetKey(demandOut, reader.Names[i])
			if !demand.IsNone(demandValueOut) {
				field := typ.Fields[i]
				fields = append(fields, zed.Field{
					Name: field.Name,
					Type: typeAfterDemand(context, fieldReader.Values, demandValueOut, field.Type),
				})
			}
		}
		result, err := context.LookupTypeRecord(fields)
		if err != nil {
			// This should be unreachable - any subset of a valid type is also valid.
			panic(err)
		}
		return result

	case *vngvector.UnionReader:
		typ := typ.(*zed.TypeUnion)
		types := make([]zed.Type, 0, len(typ.Types))
		for i, unionReader := range reader.Readers {
			types = append(types, typeAfterDemand(context, unionReader, demandOut, typ.Types[i]))
		}
		return context.LookupTypeUnion(types)

	default:
		return typ
	}
}
