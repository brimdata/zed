package vng_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/brimdata/zed/zson"
)

type mockFile struct {
	bytes.Buffer
}

func (f *mockFile) Close() error { return nil }

func FuzzVngRoundtrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		bytesReader := bytes.NewReader(b)
		context := zed.NewContext()
		types := genTypes(bytesReader, context)
		values := genValues(bytesReader, types)
		roundtrip(t, values)
	})
}

func roundtrip(t *testing.T, valuesIn []zed.Value) {
	// Write
	var fileIn mockFile
	writer, err := vngio.NewWriter(&fileIn, vngio.WriterOpts{ColumnThresh: vngio.DefaultColumnThresh, SkewThresh: vngio.DefaultSkewThresh})
	if err != nil {
		t.Errorf("%v", err)
	}
	for i := range valuesIn {
		err := writer.Write(&valuesIn[i])
		if err != nil {
			t.Errorf("%v", err)
		}
	}
	err = writer.Close()
	if err != nil {
		t.Errorf("%v", err)
	}

	// Read
	fileOut := bytes.NewReader(fileIn.Bytes())
	context := zed.NewContext()
	reader, err := vngio.NewReader(context, fileOut)
	if err != nil {
		t.Errorf("%v", err)
	}
	valuesOut := make([]zed.Value, 0, len(valuesIn))
	for {
		value, err := reader.Read()
		if err != nil {
			t.Errorf("%v", err)
		}
		if value == nil {
			break
		}
		valuesOut = append(valuesOut, *(value.Copy()))
	}

	// Compare
	t.Logf("comparing: len(in)=%v vs len(out)=%v", len(valuesIn), len(valuesOut))
	for i := range valuesIn {
		if i >= len(valuesOut) {
			t.Errorf("missing value: in[%v].Bytes()=%v", i, valuesIn[i].Bytes())
			t.Errorf("missing value: in[%v]=%v", i, zson.String(&valuesIn[i]))
			continue
		}
		valueIn := valuesIn[i]
		valueOut := valuesOut[i]
		t.Logf("comparing: in[%v]=%v vs out[%v]=%v", i, zson.String(&valueIn), i, zson.String(&valueOut))
		if !bytes.Equal(zed.EncodeTypeValue(valueIn.Type), zed.EncodeTypeValue(valueOut.Type)) {
			t.Errorf("values have different types: %v %v", valueIn.Type, valueOut.Type)
		}
		if !bytes.Equal(valueIn.Bytes(), valueOut.Bytes()) {
			t.Errorf("values have different zng bytes: %v %v", valueIn.Bytes(), valueOut.Bytes())
		}
	}
	for i := range valuesOut[len(valuesIn):] {
		t.Errorf("extra value: out[%v].Bytes()=%v", i, valuesOut[i].Bytes())
		t.Errorf("extra value: out[%v]=%v", i, zson.String(&valuesOut[i]))
	}
}

func genValues(b *bytes.Reader, types []zed.Type) []zed.Value {
	values := make([]zed.Value, 0)
	for genByte(b) != 0 {
		values = append(values, *genValue(b, types))
	}
	return values
}

func genValue(b *bytes.Reader, types []zed.Type) *zed.Value {
	typ := types[int(genByte(b))%len(types)]
	if genByte(b) == 0 {
		return zed.NewValue(typ, nil)
	}
	switch typ {
	case zed.TypeUint8:
		return zed.NewUint8(genByte(b))
	case zed.TypeUint16:
		return zed.NewUint16(binary.LittleEndian.Uint16(genBytes(b, 2)))
	case zed.TypeUint32:
		return zed.NewUint32(binary.LittleEndian.Uint32(genBytes(b, 4)))
	case zed.TypeUint64:
		return zed.NewUint64(binary.LittleEndian.Uint64(genBytes(b, 8)))
	case zed.TypeInt8:
		return zed.NewInt8(int8(genByte(b)))
	case zed.TypeInt16:
		return zed.NewInt16(int16(binary.LittleEndian.Uint16(genBytes(b, 2))))
	case zed.TypeInt32:
		return zed.NewInt32(int32(binary.LittleEndian.Uint32(genBytes(b, 4))))
	case zed.TypeInt64:
		return zed.NewInt64(int64(binary.LittleEndian.Uint64(genBytes(b, 8))))
	case zed.TypeDuration:
		return zed.NewDuration(nano.Duration(int64(binary.LittleEndian.Uint64(genBytes(b, 8)))))
	case zed.TypeTime:
		return zed.NewTime(nano.Ts(int64(binary.LittleEndian.Uint64(genBytes(b, 8)))))
	case zed.TypeFloat16:
		panic("Unreachable")
	case zed.TypeFloat32:
		return zed.NewFloat32(math.Float32frombits(binary.LittleEndian.Uint32(genBytes(b, 4))))
	case zed.TypeFloat64:
		return zed.NewFloat64(math.Float64frombits(binary.LittleEndian.Uint64(genBytes(b, 8))))
	case zed.TypeBool:
		return zed.NewBool(genByte(b) > 0)
	case zed.TypeBytes:
		return zed.NewBytes(genBytes(b, int(genByte(b))))
	case zed.TypeString:
		return zed.NewString(string(genBytes(b, int(genByte(b)))))
	case zed.TypeIP, zed.TypeNet, zed.TypeType:
		panic("Unreachable")
	case zed.TypeNull:
		return zed.Null
	default:
		switch typ := typ.(type) {
		case *zed.TypeRecord:
			var builder zcode.Builder
			builder.BeginContainer()
			for _, field := range typ.Fields {
				value := genValue(b, []zed.Type{field.Type})
				builder.Append(value.Bytes())
			}
			builder.EndContainer()
			return zed.NewValue(typ, builder.Bytes())
		case *zed.TypeArray:
			elems := genValues(b, []zed.Type{typ.Type})
			var builder zcode.Builder
			builder.BeginContainer()
			for _, elem := range elems {
				builder.Append(elem.Bytes())
			}
			builder.EndContainer()
			return zed.NewValue(typ, builder.Bytes())
		case *zed.TypeMap:
			var builder zcode.Builder
			builder.BeginContainer()
			for genByte(b) != 0 {
				builder.Append(genValue(b, []zed.Type{typ.KeyType}).Bytes())
				builder.Append(genValue(b, []zed.Type{typ.ValType}).Bytes())
			}
			builder.TransformContainer(zed.NormalizeMap)
			builder.EndContainer()
			return zed.NewValue(typ, builder.Bytes())
		case *zed.TypeSet:
			elems := genValues(b, []zed.Type{typ.Type})
			var builder zcode.Builder
			builder.BeginContainer()
			for _, elem := range elems {
				builder.Append(elem.Bytes())
			}
			builder.TransformContainer(zed.NormalizeSet)
			builder.EndContainer()
			return zed.NewValue(typ, builder.Bytes())
		// TODO TypeUnion
		default:
			panic("Unreachable")
		}
	}
}

func genTypes(b *bytes.Reader, context *zed.Context) []zed.Type {
	types := make([]zed.Type, 0)
	for len(types) == 0 || genByte(b) != 0 {
		types = append(types, genType(b, context))
	}
	return types
}

func genType(b *bytes.Reader, context *zed.Context) zed.Type {
	switch genByte(b) % 23 {
	case 0:
		return zed.TypeUint8
	case 1:
		return zed.TypeUint16
	case 2:
		return zed.TypeUint32
	case 3:
		return zed.TypeUint64
	case 4:
		return zed.TypeInt8
	case 5:
		return zed.TypeInt16
	case 6:
		return zed.TypeInt32
	case 7:
		return zed.TypeInt64
	case 8:
		return zed.TypeDuration
	case 9:
		return zed.TypeTime
	case 10:
		// TODO Find a way to convert u16 to float16.
		//return zed.TypeFloat16
		return zed.TypeNull
	case 11:
		return zed.TypeFloat32
	case 12:
		return zed.TypeBool
	case 13:
		return zed.TypeBytes
	case 14:
		return zed.TypeString
	case 15:
		// TODO
		//return zed.TypeIP
		return zed.TypeNull
	case 16:
		// TODO
		//return zed.TypeNet
		return zed.TypeNull
	case 17:
		// TODO
		//return zed.TypeType
		return zed.TypeNull
	case 18:
		return zed.TypeNull
	case 19:
		fieldTypes := genTypes(b, context)
		fields := make([]zed.Field, len(fieldTypes))
		for i, fieldType := range fieldTypes {
			fields[i] = zed.Field{
				Name: fmt.Sprint(i),
				Type: fieldType,
			}
		}
		typ, err := context.LookupTypeRecord(fields)
		if err != nil {
			panic(err)
		}
		return typ
	case 20:
		elem := genType(b, context)
		return context.LookupTypeArray(elem)
	case 21:
		key := genType(b, context)
		value := genType(b, context)
		return context.LookupTypeMap(key, value)
	case 22:
		elem := genType(b, context)
		return context.LookupTypeSet(elem)
		// TODO TypeUnion
	default:
		panic("Unreachable")
	}
}

func genByte(b *bytes.Reader) byte {
	// If we're out of bytes, return 0.
	byte, _ := b.ReadByte()
	return byte
}

func genBytes(b *bytes.Reader, n int) []byte {
	bytes := make([]byte, n)
	for i := range bytes {
		bytes[i] = genByte(b)
	}
	return bytes
}
