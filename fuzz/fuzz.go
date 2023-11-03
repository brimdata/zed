package fuzz

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
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage/mock"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

func WriteZng(t *testing.T, valuesIn []zed.Value, file *MockFile) {
	writer := zngio.NewWriter(file)
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
}

func WriteVng(t *testing.T, valuesIn []zed.Value, file *MockFile) {
	writer, err := vngio.NewWriter(file)
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
}

func RunQueryZng(t *testing.T, file *MockFile, querySource string) []zed.Value {
	zctx := zed.NewContext()
	readers := []zio.Reader{zngio.NewReader(zctx, bytes.NewReader(file.Bytes()))}
	defer zio.CloseReaders(readers)
	return RunQuery(t, zctx, readers, querySource, func(_ demand.Demand) {})
}

func RunQueryVng(t *testing.T, file *MockFile, querySource string) []zed.Value {
	zctx := zed.NewContext()
	reader, err := vngio.NewReader(zctx, bytes.NewReader(file.Bytes()))
	if err != nil {
		t.Fatalf("%v", err)
	}
	readers := []zio.Reader{reader}
	defer zio.CloseReaders(readers)
	return RunQuery(t, zctx, readers, querySource, func(demandIn demand.Demand) {
		if reader, ok := readers[0].(*vngio.Reader); ok {
			reader.Opts.Demand = demandIn
		}
	})
}

func RunQuery(t *testing.T, zctx *zed.Context, readers []zio.Reader, querySource string, useDemand func(demandIn demand.Demand)) []zed.Value {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Compile query
	engine := mock.NewMockEngine(gomock.NewController(t))
	comp := compiler.NewFileSystemCompiler(engine)
	ast, err := compiler.Parse(querySource)
	if err != nil {
		t.Skipf("%v", err)
	}
	query, err := runtime.CompileQuery(ctx, zctx, comp, ast, readers)
	if err != nil {
		t.Skipf("%v", err)
	}
	defer query.Pull(true)

	// Infer demand
	// TODO This is a hack and should be replaced by a cleaner interface in CompileQuery.
	source := data.NewSource(engine, nil)
	dag, err := semantic.AnalyzeAddSource(ctx, ast, source, nil)
	if err != nil {
		t.Skipf("%v", err)
	}
	if len(dag) > 0 {
		demands := optimizer.InferDemandSeqOut(dag)
		demand := demands[dag[0]]
		useDemand(demand)
	}

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

func CompareValues(t *testing.T, valuesExpected []zed.Value, valuesActual []zed.Value) {
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

type MockFile struct {
	bytes.Buffer
}

func (f *MockFile) Close() error { return nil }

func GenValues(b *bytes.Reader, context *zed.Context, types []zed.Type) []zed.Value {
	values := make([]zed.Value, 0)
	var builder zcode.Builder
	for GenByte(b) != 0 {
		typ := types[int(GenByte(b))%len(types)]
		builder.Reset()
		GenValue(b, context, typ, &builder)
		it := builder.Bytes().Iter()
		values = append(values, *zed.NewValue(typ, it.Next()).Copy())
	}
	return values
}

func GenValue(b *bytes.Reader, context *zed.Context, typ zed.Type, builder *zcode.Builder) {
	if GenByte(b) == 0 {
		builder.Append(nil)
		return
	}
	switch typ {
	case zed.TypeUint8:
		builder.Append(zed.EncodeUint(uint64(GenByte(b))))
	case zed.TypeUint16:
		builder.Append(zed.EncodeUint(uint64(binary.LittleEndian.Uint16(GenBytes(b, 2)))))
	case zed.TypeUint32:
		builder.Append(zed.EncodeUint(uint64(binary.LittleEndian.Uint32(GenBytes(b, 4)))))
	case zed.TypeUint64:
		builder.Append(zed.EncodeUint(uint64(binary.LittleEndian.Uint64(GenBytes(b, 8)))))
	case zed.TypeInt8:
		builder.Append(zed.EncodeInt(int64(GenByte(b))))
	case zed.TypeInt16:
		builder.Append(zed.EncodeInt(int64(binary.LittleEndian.Uint16(GenBytes(b, 2)))))
	case zed.TypeInt32:
		builder.Append(zed.EncodeInt(int64(binary.LittleEndian.Uint32(GenBytes(b, 4)))))
	case zed.TypeInt64:
		builder.Append(zed.EncodeInt(int64(binary.LittleEndian.Uint64(GenBytes(b, 8)))))
	case zed.TypeDuration:
		builder.Append(zed.EncodeDuration(nano.Duration(int64(binary.LittleEndian.Uint64(GenBytes(b, 8))))))
	case zed.TypeTime:
		builder.Append(zed.EncodeTime(nano.Ts(int64(binary.LittleEndian.Uint64(GenBytes(b, 8))))))
	case zed.TypeFloat16:
		panic("Unreachable")
	case zed.TypeFloat32:
		builder.Append(zed.EncodeFloat32(math.Float32frombits(binary.LittleEndian.Uint32(GenBytes(b, 4)))))
	case zed.TypeFloat64:
		builder.Append(zed.EncodeFloat64(math.Float64frombits(binary.LittleEndian.Uint64(GenBytes(b, 8)))))
	case zed.TypeBool:
		builder.Append(zed.EncodeBool(GenByte(b) > 0))
	case zed.TypeBytes:
		builder.Append(zed.EncodeBytes(GenBytes(b, int(GenByte(b)))))
	case zed.TypeString:
		builder.Append(zed.EncodeString(string(GenBytes(b, int(GenByte(b))))))
	case zed.TypeIP:
		builder.Append(zed.EncodeIP(netip.AddrFrom16(*(*[16]byte)(GenBytes(b, 16)))))
	case zed.TypeNet:
		ip := netip.AddrFrom16(*(*[16]byte)(GenBytes(b, 16)))
		numBits := int(GenByte(b)) % ip.BitLen()
		net, err := ip.Prefix(numBits)
		if err != nil {
			// Should be unreachable.
			panic(err)
		}
		builder.Append(zed.EncodeNet(net))
	case zed.TypeType:
		typ := GenType(b, context, 3)
		builder.Append(zed.EncodeTypeValue(typ))
	case zed.TypeNull:
		builder.Append(nil)
	default:
		switch typ := typ.(type) {
		case *zed.TypeRecord:
			builder.BeginContainer()
			for _, field := range typ.Fields {
				GenValue(b, context, field.Type, builder)
			}
			builder.EndContainer()
		case *zed.TypeArray:
			builder.BeginContainer()
			for GenByte(b) != 0 {
				GenValue(b, context, typ.Type, builder)
			}
			builder.EndContainer()
		case *zed.TypeMap:
			builder.BeginContainer()
			for GenByte(b) != 0 {
				GenValue(b, context, typ.KeyType, builder)
				GenValue(b, context, typ.ValType, builder)
			}
			builder.TransformContainer(zed.NormalizeMap)
			builder.EndContainer()
		case *zed.TypeSet:
			builder.BeginContainer()
			for GenByte(b) != 0 {
				GenValue(b, context, typ.Type, builder)
			}
			builder.TransformContainer(zed.NormalizeSet)
			builder.EndContainer()
		case *zed.TypeUnion:
			tag := binary.LittleEndian.Uint64(GenBytes(b, 8)) % uint64(len(typ.Types))
			builder.BeginContainer()
			builder.Append(zed.EncodeInt(int64(tag)))
			GenValue(b, context, typ.Types[tag], builder)
			builder.EndContainer()
		default:
			panic("Unreachable")
		}
	}
}

func GenTypes(b *bytes.Reader, context *zed.Context, depth int) []zed.Type {
	types := make([]zed.Type, 0)
	for len(types) == 0 || GenByte(b) != 0 {
		types = append(types, GenType(b, context, depth))
	}
	return types
}

func GenType(b *bytes.Reader, context *zed.Context, depth int) zed.Type {
	if depth < 0 || GenByte(b)%2 == 0 {
		switch GenByte(b) % 19 {
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
		switch GenByte(b) % 5 {
		case 0:
			fieldTypes := GenTypes(b, context, depth)
			fields := make([]zed.Field, len(fieldTypes))
			for i, fieldType := range fieldTypes {
				fields[i] = zed.Field{
					Name: fmt.Sprintf("f%d", i),
					Type: fieldType,
				}
			}
			typ, err := context.LookupTypeRecord(fields)
			if err != nil {
				panic(err)
			}
			return typ
		case 1:
			elem := GenType(b, context, depth)
			return context.LookupTypeArray(elem)
		case 2:
			key := GenType(b, context, depth)
			value := GenType(b, context, depth)
			return context.LookupTypeMap(key, value)
		case 3:
			elem := GenType(b, context, depth)
			return context.LookupTypeSet(elem)
		case 4:
			types := GenTypes(b, context, depth)
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

func GenByte(b *bytes.Reader) byte {
	// If we're out of bytes, return 0.
	byte, _ := b.ReadByte()
	return byte
}

func GenBytes(b *bytes.Reader, n int) []byte {
	bytes := make([]byte, n)
	for i := range bytes {
		bytes[i] = GenByte(b)
	}
	return bytes
}

func GenAscii(b *bytes.Reader) string {
	bytes := make([]byte, 0)
	for {
		byte := GenByte(b)
		if byte == 0 {
			break
		}
		bytes = append(bytes, byte)
	}
	return string(bytes)
}
