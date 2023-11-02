package vector

import (
	"bytes"
	"fmt"
	"io"
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/vng"
	vngvector "github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

func Read(reader *vng.Reader) (*Vector, error) {
	context := zed.NewContext()
	tags, err := readInt64s(reader.Root)
	if err != nil {
		return nil, err
	}
	types := make([]zed.Type, len(reader.Readers))
	values := make([]vector, len(reader.Readers))
	for i, typedReader := range reader.Readers {
		typ, _ := context.DecodeTypeValue(zed.EncodeTypeValue(typedReader.Type))
		types[i] = typ
		value, err := read(context, typedReader.Reader)
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

func read(context *zed.Context, reader vngvector.Reader) (vector, error) {
	switch reader := reader.(type) {

	case *vngvector.ArrayReader:
		lengths, err := readInt64s(reader.Lengths)
		if err != nil {
			return nil, err
		}
		elems, err := read(context, reader.Elems)
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
		value := zed.NewValue(reader.Typ, it.Next())
		vector := &constants{
			value: *value,
		}
		return vector, nil

	case *vngvector.DictReader:
		// TODO Would we be better off with a dicts vector?
		return readPrimitive(context, reader.Typ, func() ([]byte, error) { return reader.ReadBytes() })

	case *vngvector.MapReader:
		keys, err := read(context, reader.Keys)
		if err != nil {
			return nil, err
		}
		lengths, err := readInt64s(reader.Lengths)
		if err != nil {
			return nil, err
		}
		values, err := read(context, reader.Values)
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
		runs, err := readInt64s(&reader.Runs)
		if err != nil {
			return nil, err
		}
		values, err := read(context, reader.Values)
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
		return readPrimitive(context, reader.Typ, func() ([]byte, error) { return reader.ReadBytes() })

	case vngvector.RecordReader: // Not a typo - RecordReader does not have a pointer receiver.
		fields := make([]vector, len(reader))
		for i, fieldReader := range reader {
			field, err := read(context, fieldReader.Values)
			if err != nil {
				return nil, err
			}
			fields[i] = field
		}
		vector := &records{
			fields: fields,
		}
		return vector, nil

	case *vngvector.UnionReader:
		payloads := make([]vector, len(reader.Readers))
		for i, reader := range reader.Readers {
			payload, err := read(context, reader)
			if err != nil {
				return nil, err
			}
			payloads[i] = payload
		}
		tags, err := readInt64s(reader.Tags)
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

// TODO This is likely to be a bottleneck. If so, inline `readBytes` and `zed.Decode*`.
func readPrimitive(context *zed.Context, typ zed.Type, readBytes func() ([]byte, error)) (vector, error) {
	switch typ {
	case zed.TypeBool:
		values := make([]bool, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeBool(bs))
		}
		vector := &bools{
			values: values,
		}
		return vector, nil

	case zed.TypeBytes:
		data := bytes.NewBuffer(nil)
		offsets := make([]int, 1)
		offsets[0] = 0
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			data.Write(zed.DecodeBytes(bs))
			offsets = append(offsets, data.Len())
		}
		vector := &byteses{
			data: data.Bytes(),
			// TODO truncate offsets
			offsets: offsets,
		}
		return vector, nil

	case zed.TypeDuration:
		values := make([]nano.Duration, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeDuration(bs))
		}
		vector := &durations{
			values: values,
		}
		return vector, nil

	case zed.TypeFloat16:
		values := make([]float32, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeFloat16(bs))
		}
		vector := &float16s{
			values: values,
		}
		return vector, nil

	case zed.TypeFloat32:
		values := make([]float32, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeFloat32(bs))
		}
		vector := &float32s{
			values: values,
		}
		return vector, nil

	case zed.TypeFloat64:
		values := make([]float64, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeFloat64(bs))
		}
		vector := &float64s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt8:
		values := make([]int8, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, int8(zed.DecodeInt(bs)))
		}
		vector := &int8s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt16:
		values := make([]int16, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, int16(zed.DecodeInt(bs)))
		}
		vector := &int16s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt32:
		values := make([]int32, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, int32(zed.DecodeInt(bs)))
		}
		vector := &int32s{
			values: values,
		}
		return vector, nil

	case zed.TypeInt64:
		values := make([]int64, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, int64(zed.DecodeInt(bs)))
		}
		vector := &int64s{
			values: values,
		}
		return vector, nil

	case zed.TypeIP:
		values := make([]netip.Addr, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeIP(bs))
		}
		vector := &ips{
			values: values,
		}
		return vector, nil

	case zed.TypeNet:
		values := make([]netip.Prefix, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeNet(bs))
		}
		vector := &nets{
			values: values,
		}
		return vector, nil

	case zed.TypeString:
		data := bytes.NewBuffer(nil)
		offsets := make([]int, 1)
		offsets[0] = 0
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			data.Write(zed.DecodeBytes(bs))
			offsets = append(offsets, data.Len())
		}
		vector := &strings{
			data: data.Bytes(),
			// TODO truncate offsets
			offsets: offsets,
		}
		return vector, nil

	case zed.TypeTime:
		values := make([]nano.Ts, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeTime(bs))
		}
		vector := &times{
			values: values,
		}
		return vector, nil

	case zed.TypeUint8:
		values := make([]uint8, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, uint8(zed.DecodeUint(bs)))
		}
		vector := &uint8s{
			values: values,
		}
		return vector, nil

	case zed.TypeUint16:
		values := make([]uint16, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, uint16(zed.DecodeUint(bs)))
		}
		vector := &uint16s{
			values: values,
		}
		return vector, nil

	case zed.TypeUint32:
		values := make([]uint32, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, uint32(zed.DecodeUint(bs)))
		}
		vector := &uint32s{
			values: values,
		}
		return vector, nil

	case zed.TypeUint64:
		values := make([]uint64, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, uint64(zed.DecodeUint(bs)))
		}
		vector := &uint64s{
			values: values,
		}
		return vector, nil

	case zed.TypeNull:
		vector := &constants{
			value: *zed.NewValue(zed.TypeNull, nil),
		}
		return vector, nil

	case zed.TypeType:
		values := make([]zed.Type, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			typ, _ := context.DecodeTypeValue(bs)
			values = append(values, typ)
		}
		vector := &types{
			values: values,
		}
		return vector, nil

	default:
		return nil, fmt.Errorf("unknown VNG type: %T", typ)
	}
}

func readInt64s(reader *vngvector.Int64Reader) ([]int64, error) {
	ints := make([]int64, 0)
	for {
		int, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		ints = append(ints, int)
	}
	return ints, nil
}
