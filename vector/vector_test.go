package vector_test

import (
	"bytes"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/fuzz"
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
