package vector

import (
	"bytes"
	"fmt"
	"io"
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/vng"
	vngVector "github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"

	"github.com/RoaringBitmap/roaring"
)

func Read(reader *vng.Reader) (*Vector, error) {
	tags, err := readInt64s(reader.Root)
	if err != nil {
		return nil, err
	}
	types := make([]zed.Type, len(reader.Readers))
	values := make([]any, len(reader.Readers))
	for i, typedReader := range reader.Readers {
		types[i] = typedReader.Typ
		value, err := read(typedReader.Reader)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	vector := &Vector{
		Types:  types,
		values: values,
		tags:   tags,
	}
	return vector, nil
}

func read(reader vngVector.Reader) (any, error) {
	switch reader := reader.(type) {

	case *vngVector.ArrayReader:
		lengths, err := readInt64s(reader.Lengths)
		if err != nil {
			return nil, err
		}
		elems, err := read(reader.Elems)
		if err != nil {
			return nil, err
		}
		vector := &arrays{
			lengths: lengths,
			elems:   elems,
		}
		return vector, nil

	case *vngVector.ConstReader:
		var builder zcode.Builder
		err := reader.Read(&builder)
		if err != nil {
			return nil, err
		}
		value := zed.NewValue(reader.Typ, builder.Bytes())
		vector := &constants{
			value: *value,
		}
		return vector, nil

	case *vngVector.DictReader:
		// TODO Would we be better off with a dicts vector?
		return readPrimitive(reader.Typ, func() ([]byte, error) { return reader.ReadBytes() })

	case *vngVector.MapReader:
		keys, err := read(reader.Keys)
		if err != nil {
			return nil, err
		}
		lengths, err := readInt64s(reader.Lengths)
		if err != nil {
			return nil, err
		}
		values, err := read(reader.Values)
		if err != nil {
			return nil, err
		}
		vector := &maps{
			lengths: lengths,
			keys:    keys,
			values:  values,
		}
		return vector, nil

	case *vngVector.NullsReader:
		mask := roaring.New()
		var maskIndex uint64
		maskBool := true
		for {
			run, err := reader.Runs.Read()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			if maskBool {
				mask.AddRange(maskIndex, maskIndex+uint64(run))
			}
			maskBool = !maskBool
			maskIndex += uint64(run)
		}
		values, err := read(reader.Values)
		if err != nil {
			return nil, err
		}
		vector := &nulls{
			mask:   mask,
			values: values,
		}
		return vector, nil

	case *vngVector.PrimitiveReader:
		return readPrimitive(reader.Typ, func() ([]byte, error) { return reader.ReadBytes() })

	case vngVector.RecordReader: // Not a typo - RecordReader does not have a pointer receiver.
		fields := make([]any, len(reader))
		for i, fieldReader := range reader {
			field, err := read(fieldReader.Values)
			if err != nil {
				return nil, err
			}
			fields[i] = field
		}
		vector := &records{
			fields: fields,
		}
		return vector, nil

	case *vngVector.UnionReader:
		payloads := make([]any, len(reader.Readers))
		for i, reader := range reader.Readers {
			payload, err := read(reader)
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
func readPrimitive(typ zed.Type, readBytes func() ([]byte, error)) (any, error) {
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
		values := make([][]byte, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeBytes(bytes.Clone(bs)))
		}
		vector := &byteses{
			values: values,
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

	case zed.TypeInt8, zed.TypeInt16, zed.TypeInt32, zed.TypeInt64:
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
			values = append(values, zed.DecodeInt(bs))
		}
		vector := &ints{
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
		values := make([]string, 0)
		for {
			bs, err := readBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeString(bytes.Clone(bs)))
		}
		vector := &strings{
			values: values,
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

	case zed.TypeUint8, zed.TypeUint16, zed.TypeUint32, zed.TypeUint64:
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
			values = append(values, zed.DecodeUint(bs))
		}
		vector := &uints{
			values: values,
		}
		return vector, nil

	case zed.TypeNull:
		vector := &constants{
			value: *zed.NewValue(zed.TypeNull, nil),
		}
		return vector, nil

	case zed.TypeType, zed.TypeNull:
		return nil, fmt.Errorf("TODO vector.read: %T", typ)

	default:
		return nil, fmt.Errorf("unknown VNG type: %T", typ)
	}
}

func readInt64s(reader *vngVector.Int64Reader) ([]int64, error) {
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
