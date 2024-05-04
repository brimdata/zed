package vector_test

import (
	"bytes"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/fuzz"
)

func FuzzQuery(f *testing.F) {
	f.Add([]byte("yield f1\x00"))
	f.Add([]byte("yield f1, f2\x00"))
	f.Add([]byte("f1 == null\x00"))
	f.Add([]byte("f1 == null | yield f2\x00"))
	f.Fuzz(func(t *testing.T, b []byte) {
		bytesReader := bytes.NewReader(b)
		querySource := fuzz.GenAscii(bytesReader)
		zctx := zed.NewContext()
		types := fuzz.GenTypes(bytesReader, zctx, 3)
		batch := fuzz.GenValues(bytesReader, zctx, types)
		defer batch.Unref()

		// Debug
		//for i := range values {
		//    t.Logf("value: in[%v].Bytes()=%v", i, values[i].Bytes())
		//    t.Logf("value: in[%v]=%v", i, zson.String(&values[i]))
		//}

		var zngBuf bytes.Buffer
		fuzz.WriteZNG(t, batch, &zngBuf)
		resultZNG := fuzz.RunQueryZNG(t, &zngBuf, querySource)
		defer resultZNG.Unref()

		var vngBuf bytes.Buffer
		fuzz.WriteVNG(t, batch, &vngBuf)
		resultVNG := fuzz.RunQueryVNG(t, &vngBuf, querySource)
		defer resultVNG.Unref()

		fuzz.CompareValues(t, resultZNG.Values(), resultVNG.Values())
	})
}
