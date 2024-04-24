package vng_test

import (
	"bytes"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/fuzz"
	"github.com/brimdata/zed/zbuf"
	"github.com/stretchr/testify/require"
)

func FuzzVngRoundtripGen(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		bytesReader := bytes.NewReader(b)
		context := zed.NewContext()
		types := fuzz.GenTypes(bytesReader, context, 3)
		batch := fuzz.GenValues(bytesReader, context, types)
		defer batch.Unref()
		roundtrip(t, batch)
	})
}

func FuzzVngRoundtripBytes(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		values, err := fuzz.ReadZNG(b)
		if err != nil {
			t.Skipf("%v", err)
		}
		roundtrip(t, values)
	})
}

func roundtrip(t *testing.T, batch zbuf.Batch) {
	var buf bytes.Buffer
	fuzz.WriteVNG(t, batch, &buf)
	batchOut, err := fuzz.ReadVNG(buf.Bytes(), demand.All())
	require.NoError(t, err)
	fuzz.CompareValues(t, batch.Values(), batchOut.Values())
}
