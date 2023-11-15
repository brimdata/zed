package vector_test

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/fuzz"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio/vngio"
)

func FuzzQuery(f *testing.F) {
	f.Add([]byte("yield f1\x00"))
	f.Add([]byte("yield f1, f2\x00"))
	f.Add([]byte("f1 == null\x00"))
	f.Add([]byte("f1 == null | yield f2\x00"))
	f.Fuzz(func(t *testing.T, b []byte) {
		bytesReader := bytes.NewReader(b)
		querySource := fuzz.GenAscii(bytesReader)
		context := zed.NewContext()
		types := fuzz.GenTypes(bytesReader, context, 3)
		values := fuzz.GenValues(bytesReader, context, types)

		// Debug
		//for i := range values {
		//    t.Logf("value: in[%v].Bytes()=%v", i, values[i].Bytes())
		//    t.Logf("value: in[%v]=%v", i, zson.String(&values[i]))
		//}

		var zngBuf bytes.Buffer
		fuzz.WriteZNG(t, values, &zngBuf)
		resultZNG := fuzz.RunQueryZNG(t, &zngBuf, querySource)

		var vngBuf bytes.Buffer
		fuzz.WriteVNG(t, values, &vngBuf, vngio.WriterOpts{
			SkewThresh:   vngio.DefaultSkewThresh,
			ColumnThresh: vngio.DefaultColumnThresh,
		})
		resultVNG := fuzz.RunQueryVNG(t, &vngBuf, querySource)

		fuzz.CompareValues(t, resultZNG, resultVNG)
	})
}

const N = 10000000

func BenchmarkReadZng(b *testing.B) {
	rand := rand.New(rand.NewSource(42))
	valuesIn := make([]zed.Value, N)
	for i := range valuesIn {
		valuesIn[i] = *zed.NewInt64(rand.Int63n(N))
	}
	var buf bytes.Buffer
	fuzz.WriteZNG(b, valuesIn, &buf)
	bs := buf.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valuesOut, err := fuzz.ReadZNG(bs)
		if err != nil {
			panic(err)
		}
		if zed.DecodeInt(valuesIn[N-1].Bytes()) != zed.DecodeInt(valuesOut[N-1].Bytes()) {
			panic("oh no")
		}
	}
}

func BenchmarkReadVng(b *testing.B) {
	rand := rand.New(rand.NewSource(42))
	valuesIn := make([]zed.Value, N)
	for i := range valuesIn {
		valuesIn[i] = *zed.NewValue(zed.TypeInt64, zed.EncodeInt(int64(rand.Intn(N))))
	}
	var buf bytes.Buffer
	fuzz.WriteVNG(b, valuesIn, &buf, vngio.WriterOpts{
		SkewThresh:   vngio.DefaultSkewThresh,
		ColumnThresh: vngio.DefaultColumnThresh,
	})
	bs := buf.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesReader := bytes.NewReader(bs)
		context := zed.NewContext()
		object, err := vng.NewObject(context, bytesReader, int64(len(bs)))
		if err != nil {
			panic(err)
		}
		vector, err := vector.Read(object, demand.All())
		if err != nil {
			panic(err)
		}
		// TODO Expose a cheap way to get values out of vectors.
		//if intsIn[N-1] != intsOut[N-1] {
		//    panic("oh no")
		//}
		_ = vector
	}
}

func BenchmarkReadVarint(b *testing.B) {
	rand := rand.New(rand.NewSource(42))
	intsIn := make([]int64, N)
	for i := range intsIn {
		intsIn[i] = int64(rand.Intn(N))
	}
	var bs []byte
	for _, int := range intsIn {
		bs = binary.AppendVarint(bs, int)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs := bs
		intsOut := make([]int64, N)
		for i := range intsOut {
			value, n := binary.Varint(bs)
			if n <= 0 {
				panic("oh no")
			}
			bs = bs[n:]
			intsOut[i] = value
		}
		if intsIn[N-1] != intsOut[N-1] {
			panic("oh no")
		}
	}
}
