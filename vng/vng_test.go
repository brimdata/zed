package vng_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/spf13/afero"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/brimdata/zed/zson"
)

func FuzzVngRoundtrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		valuesIn := genValues(bytes.NewReader(b))

		var Fs = afero.NewMemMapFs()

		// Write
		file, err := Fs.Create("test.vng")
		if err != nil {
			t.Errorf("%v", err)
		}
		writer, err := vngio.NewWriter(file, vngio.WriterOpts{ColumnThresh: vngio.DefaultColumnThresh, SkewThresh: vngio.DefaultSkewThresh})
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
		file, err = Fs.Open("test.vng")
		if err != nil {
			t.Errorf("%v", err)
		}
		context := zed.NewContext()
		reader, err := vngio.NewReader(context, file)
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
			valuesOut = append(valuesOut, *value)
		}

		// Compare
		t.Logf("comparing: len(in)=%v vs len(out)=%v", len(valuesIn), len(valuesOut))
		for i := range valuesIn {
			if i >= len(valuesOut) {
				t.Errorf("missing value: %v", valuesIn[i])
			}
			valueIn := valuesIn[i]
			valueOut := valuesOut[i]
			t.Logf("comparing: in[%v]=%v vs out[%v]=%v", i, zson.String(&valueIn), i, zson.String(&valueOut))
			if valueIn.Type != valueOut.Type {
				t.Errorf("values have different types: %v %v", valueIn.Type, valueOut.Type)
			}
			if !bytes.Equal(valueIn.Bytes(), valueOut.Bytes()) {
				t.Errorf("values have different zng bytes: %v %v", valueIn.Bytes(), valueOut.Bytes())
			}
		}
		for _, value := range valuesOut[len(valuesIn):] {
			t.Errorf("extra value: %v", value)
		}
	})
}

func genValues(b *bytes.Reader) []zed.Value {
	types := genTypes(b)
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
		panic("Unreachable")
	}
}

func genTypes(b *bytes.Reader) []zed.Type {
	types := make([]zed.Type, 0)
	for len(types) == 0 || genByte(b) != 0 {
		types = append(types, genType(b))
	}
	return types
}

func genType(b *bytes.Reader) zed.Type {
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
