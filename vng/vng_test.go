package vng_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/fuzz"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/stretchr/testify/require"
)

func FuzzVngRoundtripGen(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		bytesReader := bytes.NewReader(b)
		context := zed.NewContext()
		types := fuzz.GenTypes(bytesReader, context, 3)
		values := fuzz.GenValues(bytesReader, context, types)
		ColumnThresh := int(binary.LittleEndian.Uint64(fuzz.GenBytes(bytesReader, 8)))
		if ColumnThresh == 0 {
			ColumnThresh = 1
		}
		if ColumnThresh > vng.MaxSegmentThresh {
			ColumnThresh = vng.MaxSegmentThresh
		}
		SkewThresh := int(binary.LittleEndian.Uint64(fuzz.GenBytes(bytesReader, 8)))
		if SkewThresh == 0 {
			SkewThresh = 1
		}
		if SkewThresh > vng.MaxSkewThresh {
			SkewThresh = vng.MaxSkewThresh
		}
		writerOpts := vngio.WriterOpts{
			ColumnThresh: units.Bytes(ColumnThresh),
			SkewThresh:   units.Bytes(SkewThresh),
		}
		roundtrip(t, values, writerOpts)
	})
}

func FuzzVngRoundtripBytes(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		values, err := fuzz.ReadZNG(b)
		if err != nil {
			t.Skipf("%v", err)
		}
		roundtrip(t, values, vngio.WriterOpts{
			ColumnThresh: units.Bytes(vngio.DefaultColumnThresh),
			SkewThresh:   units.Bytes(vngio.DefaultSkewThresh),
		})
	})
}

func roundtrip(t *testing.T, valuesIn []zed.Value, writerOpts vngio.WriterOpts) {
	var buf bytes.Buffer
	fuzz.WriteVNG(t, valuesIn, &buf, writerOpts)
	valuesOut, err := fuzz.ReadVNG(buf.Bytes())
	require.NoError(t, err)
	fuzz.CompareValues(t, valuesIn, valuesOut)
}
