package vector

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vng"
	vngVector "github.com/brimdata/zed/vng/vector"

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
		// Why does const have a count?
		panic("TODO")

	case *vngVector.DictReader:
		panic("TODO")

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
		maskBool := false
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
		return readPrimitive(reader)

	case *vngVector.RecordReader:
		fields := make([]any, len(*reader))
		for i, fieldReader := range *reader {
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

func readPrimitive(reader *vngVector.PrimitiveReader) (any, error) {
	switch reader.Typ {
	case zed.TypeBool:
		values := make([]bool, 0)
		for {
			bytes, err := reader.ReadBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeBool(bytes))
		}
		vector := &bools{
			values: values,
		}
		return vector, nil

	case zed.TypeUint8, zed.TypeUint16, zed.TypeUint32, zed.TypeUint64:
		values := make([]uint64, 0)
		for {
			bytes, err := reader.ReadBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeUint(bytes))
		}
		vector := &uints{
			values: values,
		}
		return vector, nil

	case zed.TypeInt8, zed.TypeInt16, zed.TypeInt32, zed.TypeInt64:
		values := make([]int64, 0)
		for {
			bytes, err := reader.ReadBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeInt(bytes))
		}
		vector := &ints{
			values: values,
		}
		return vector, nil

	case zed.TypeString:
		values := make([]string, 0)
		for {
			bytes, err := reader.ReadBytes()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			values = append(values, zed.DecodeString(bytes))
		}
		vector := &strings{
			values: values,
		}
		return vector, nil

	case zed.TypeDuration, zed.TypeTime, zed.TypeFloat16, zed.TypeFloat32, zed.TypeFloat64, zed.TypeBytes, zed.TypeIP, zed.TypeNet, zed.TypeType, zed.TypeNull:
		panic("TODO")

	default:
		return nil, fmt.Errorf("unknown VNG type: %T", reader.Typ)
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
