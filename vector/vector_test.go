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
		context := zed.NewContext()
		types := fuzz.GenTypes(bytesReader, context, 3)
		values := fuzz.GenValues(bytesReader, context, types)

		// Debug
		//for i := range values {
		//    t.Logf("value: in[%v].Bytes()=%v", i, values[i].Bytes())
		//    t.Logf("value: in[%v]=%v", i, zson.String(&values[i]))
		//}

		var zngFile fuzz.MockFile
		fuzz.WriteZng(t, values, &zngFile)
		resultZng := fuzz.RunQueryZng(t, &zngFile, querySource)

		var vngFile fuzz.MockFile
		fuzz.WriteVng(t, values, &vngFile)
		resultVng := fuzz.RunQueryVng(t, &vngFile, querySource)

		fuzz.CompareValues(t, resultZng, resultVng)
	})
}
