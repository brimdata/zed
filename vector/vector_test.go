package vector_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/golang/mock/gomock"
	"math"
	"net/netip"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage/mock"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

func FuzzQuery(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		bytesReader := bytes.NewReader(b)
		context := zed.NewContext()
		types := genTypes(bytesReader, context, 3)
		values := genValues(bytesReader, context, types)

		// Debug
		//for i := range values {
		//    t.Logf("value: in[%v].Bytes()=%v", i, values[i].Bytes())
		//    t.Logf("value: in[%v]=%v", i, zson.String(&values[i]))
		//}

		querySource := "yield this"
		resultZng := runQueryZng(t, values, querySource)
		resultVng := runQueryVng(t, values, querySource)
		compareValues(t, resultZng, resultVng)
	})
}

func runQueryVng(t *testing.T, valuesIn []zed.Value, querySource string) []zed.Value {

	// Write vng file
	var fileIn mockFile
	writer, err := vngio.NewWriter(&fileIn)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for i := range valuesIn {
		err := writer.Write(&valuesIn[i])
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compile query
	zctx := zed.NewContext()
	reader, err := vngio.NewReader(zctx, bytes.NewReader(fileIn.Bytes()))
	if err != nil {
		t.Fatalf("%v", err)
	}
	readers := []zio.Reader{reader}
	defer zio.CloseReaders(readers)

	return runQuery(t, zctx, readers, querySource)
}

func runQueryZng(t *testing.T, valuesIn []zed.Value, querySource string) []zed.Value {

	// Write zng file
	var fileIn mockFile
	writer := zngio.NewWriter(&fileIn)
	for i := range valuesIn {
		err := writer.Write(&valuesIn[i])
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	err := writer.Close()
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compile query
	zctx := zed.NewContext()
	readers := []zio.Reader{zngio.NewReader(zctx, bytes.NewReader(fileIn.Bytes()))}
	defer zio.CloseReaders(readers)

	return runQuery(t, zctx, readers, querySource)
}

func runQuery(t *testing.T, zctx *zed.Context, readers []zio.Reader, querySource string) []zed.Value {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Compile query
	mockEngine := mock.NewMockEngine(gomock.NewController(t))
	comp := compiler.NewFileSystemCompiler(mockEngine)
	flowgraph, err := compiler.Parse(querySource)
	if err != nil {
		t.Fatalf("%v", err)
	}
	query, err := runtime.CompileQuery(ctx, zctx, comp, flowgraph, readers)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer query.Pull(true)

	// Run query
	valuesOut := make([]zed.Value, 0)
	for {
		batch, err := query.Pull(false)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if batch == nil {
			break
		}
		for _, value := range batch.Values() {
			valuesOut = append(valuesOut, *(value.Copy()))
		}
		batch.Unref()
	}

	return valuesOut
}

// TODO Once https://github.com/brimdata/zed/pull/4833 merges, move the below to a shared package.

func compareValues(t *testing.T, valuesExpected []zed.Value, valuesActual []zed.Value) {
	t.Logf("comparing: len(expected)=%v vs len(actual)=%v", len(valuesExpected), len(valuesActual))
	for i := range valuesExpected {
		if i >= len(valuesActual) {
			t.Errorf("missing value: expected[%v].Bytes()=%v", i, valuesExpected[i].Bytes())
			t.Errorf("missing value: expected[%v]=%v", i, zson.String(&valuesExpected[i]))
			continue
		}
		valueExpected := valuesExpected[i]
		valueActual := valuesActual[i]
		t.Logf("comparing: expected[%v]=%v vs actual[%v]=%v", i, zson.String(&valueExpected), i, zson.String(&valueActual))
		if !bytes.Equal(zed.EncodeTypeValue(valueExpected.Type), zed.EncodeTypeValue(valueActual.Type)) {
			t.Errorf("values have different types: %v vs %v", valueExpected.Type, valueActual.Type)
		}
		if !bytes.Equal(valueExpected.Bytes(), valueActual.Bytes()) {
			t.Errorf("values have different zng bytes: %v vs %v", valueExpected.Bytes(), valueActual.Bytes())
		}
	}
	for i := range valuesActual[len(valuesExpected):] {
		t.Errorf("extra value: actual[%v].Bytes()=%v", i, valuesActual[i].Bytes())
		t.Errorf("extra value: actual[%v]=%v", i, zson.String(&valuesActual[i]))
	}
}

type mockFile struct {
	bytes.Buffer
}

func (f *mockFile) Close() error { return nil }

func genValues(b *bytes.Reader, context *zed.Context, types []zed.Type) []zed.Value {
	values := make([]zed.Value, 0)
	var builder zcode.Builder
	for genByte(b) != 0 {
		typ := types[int(genByte(b))%len(types)]
		builder.Reset()
		genValue(b, context, typ, &builder)
		it := builder.Bytes().Iter()
		values = append(values, *zed.NewValue(typ, it.Next()).Copy())
	}
	return values
}

func genValue(b *bytes.Reader, context *zed.Context, typ zed.Type, builder *zcode.Builder) {
	if genByte(b) == 0 {
		builder.Append(nil)
		return
	}
	switch typ {
	case zed.TypeUint8:
		builder.Append(zed.EncodeUint(uint64(genByte(b))))
	case zed.TypeUint16:
		builder.Append(zed.EncodeUint(uint64(binary.LittleEndian.Uint16(genBytes(b, 2)))))
	case zed.TypeUint32:
		builder.Append(zed.EncodeUint(uint64(binary.LittleEndian.Uint32(genBytes(b, 4)))))
	case zed.TypeUint64:
		builder.Append(zed.EncodeUint(uint64(binary.LittleEndian.Uint64(genBytes(b, 8)))))
	case zed.TypeInt8:
		builder.Append(zed.EncodeInt(int64(genByte(b))))
	case zed.TypeInt16:
		builder.Append(zed.EncodeInt(int64(binary.LittleEndian.Uint16(genBytes(b, 2)))))
	case zed.TypeInt32:
		builder.Append(zed.EncodeInt(int64(binary.LittleEndian.Uint32(genBytes(b, 4)))))
	case zed.TypeInt64:
		builder.Append(zed.EncodeInt(int64(binary.LittleEndian.Uint64(genBytes(b, 8)))))
	case zed.TypeDuration:
		builder.Append(zed.EncodeDuration(nano.Duration(int64(binary.LittleEndian.Uint64(genBytes(b, 8))))))
	case zed.TypeTime:
		builder.Append(zed.EncodeTime(nano.Ts(int64(binary.LittleEndian.Uint64(genBytes(b, 8))))))
	case zed.TypeFloat16:
		panic("Unreachable")
	case zed.TypeFloat32:
		builder.Append(zed.EncodeFloat32(math.Float32frombits(binary.LittleEndian.Uint32(genBytes(b, 4)))))
	case zed.TypeFloat64:
		builder.Append(zed.EncodeFloat64(math.Float64frombits(binary.LittleEndian.Uint64(genBytes(b, 8)))))
	case zed.TypeBool:
		builder.Append(zed.EncodeBool(genByte(b) > 0))
	case zed.TypeBytes:
		builder.Append(zed.EncodeBytes(genBytes(b, int(genByte(b)))))
	case zed.TypeString:
		builder.Append(zed.EncodeString(string(genBytes(b, int(genByte(b))))))
	case zed.TypeIP:
		builder.Append(zed.EncodeIP(netip.AddrFrom16(*(*[16]byte)(genBytes(b, 16)))))
	case zed.TypeNet:
		ip := netip.AddrFrom16(*(*[16]byte)(genBytes(b, 16)))
		numBits := int(genByte(b)) % ip.BitLen()
		net, err := ip.Prefix(numBits)
		if err != nil {
			// Should be unreachable.
			panic(err)
		}
		builder.Append(zed.EncodeNet(net))
	case zed.TypeType:
		typ := genType(b, context, 3)
		builder.Append(zed.EncodeTypeValue(typ))
	case zed.TypeNull:
		builder.Append(nil)
	default:
		switch typ := typ.(type) {
		case *zed.TypeRecord:
			builder.BeginContainer()
			for _, field := range typ.Fields {
				genValue(b, context, field.Type, builder)
			}
			builder.EndContainer()
		case *zed.TypeArray:
			builder.BeginContainer()
			for genByte(b) != 0 {
				genValue(b, context, typ.Type, builder)
			}
			builder.EndContainer()
		case *zed.TypeMap:
			builder.BeginContainer()
			for genByte(b) != 0 {
				genValue(b, context, typ.KeyType, builder)
				genValue(b, context, typ.ValType, builder)
			}
			builder.TransformContainer(zed.NormalizeMap)
			builder.EndContainer()
		case *zed.TypeSet:
			builder.BeginContainer()
			for genByte(b) != 0 {
				genValue(b, context, typ.Type, builder)
			}
			builder.TransformContainer(zed.NormalizeSet)
			builder.EndContainer()
		case *zed.TypeUnion:
			tag := binary.LittleEndian.Uint64(genBytes(b, 8)) % uint64(len(typ.Types))
			builder.BeginContainer()
			builder.Append(zed.EncodeInt(int64(tag)))
			genValue(b, context, typ.Types[tag], builder)
			builder.EndContainer()
		default:
			panic("Unreachable")
		}
	}
}

func genTypes(b *bytes.Reader, context *zed.Context, depth int) []zed.Type {
	types := make([]zed.Type, 0)
	for len(types) == 0 || genByte(b) != 0 {
		types = append(types, genType(b, context, depth))
	}
	return types
}

func genType(b *bytes.Reader, context *zed.Context, depth int) zed.Type {
	if depth < 0 || genByte(b)%2 == 0 {
		switch genByte(b) % 19 {
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
			return zed.TypeIP
		case 16:
			return zed.TypeNet
		case 17:
			return zed.TypeType
		case 18:
			return zed.TypeNull
		default:
			panic("Unreachable")
		}
	} else {
		depth := depth - 1
		switch genByte(b) % 5 {
		case 0:
			fieldTypes := genTypes(b, context, depth)
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
		case 1:
			elem := genType(b, context, depth)
			return context.LookupTypeArray(elem)
		case 2:
			key := genType(b, context, depth)
			value := genType(b, context, depth)
			return context.LookupTypeMap(key, value)
		case 3:
			elem := genType(b, context, depth)
			return context.LookupTypeSet(elem)
		case 4:
			types := genTypes(b, context, depth)
			// TODO There are some weird corners around unions that contain null or duplicate types eg
			// vng_test.go:107: comparing: in[0]=null((null,null)) vs out[0]=null((null,null))
			// vng_test.go:112: values have different zng bytes: [1 0] vs [2 2 0]
			unionTypes := make([]zed.Type, 0)
			for _, typ := range types {
				skip := false
				if typ == zed.TypeNull {
					skip = true
				}
				for _, unionType := range unionTypes {
					if typ == unionType {
						skip = true
					}
				}
				if !skip {
					unionTypes = append(unionTypes, typ)
				}
			}
			if len(unionTypes) == 0 {
				return zed.TypeNull
			}
			return context.LookupTypeUnion(unionTypes)
		default:
			panic("Unreachable")
		}
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
