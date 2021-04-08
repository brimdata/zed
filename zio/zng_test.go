package zio_test

//  This is really a system test dressed up as a unit test.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Output struct {
	bytes.Buffer
}

func (o *Output) Close() error {
	return nil
}

// Send logs to tzng reader -> zng writer -> zng reader -> tzng writer
func boomerang(t *testing.T, logs string, compress bool) {
	in := []byte(strings.TrimSpace(logs) + "\n")
	tzngSrc := tzngio.NewReader(bytes.NewReader(in), zson.NewContext())
	var rawzng Output
	var zngLZ4BlockSize int
	if compress {
		zngLZ4BlockSize = zngio.DefaultLZ4BlockSize
	}
	rawDst := zngio.NewWriter(&rawzng, zngio.WriterOpts{LZ4BlockSize: zngLZ4BlockSize})
	require.NoError(t, zbuf.Copy(rawDst, tzngSrc))
	require.NoError(t, rawDst.Close())

	var out Output
	rawSrc := zngio.NewReader(bytes.NewReader(rawzng.Bytes()), zson.NewContext())
	tzngDst := tzngio.NewWriter(&out)
	err := zbuf.Copy(tzngDst, rawSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

func boomerangZJSON(t *testing.T, logs string) {
	tzngSrc := tzngio.NewReader(strings.NewReader(logs), zson.NewContext())
	var zjsonOutput Output
	zjsonDst := zjsonio.NewWriter(&zjsonOutput)
	err := zbuf.Copy(zjsonDst, tzngSrc)
	require.NoError(t, err)

	var out Output
	zjsonSrc := zjsonio.NewReader(bytes.NewReader(zjsonOutput.Bytes()), zson.NewContext())
	tzngDst := tzngio.NewWriter(&out)
	err = zbuf.Copy(tzngDst, zjsonSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, strings.TrimSpace(logs), strings.TrimSpace(out.String()))
	}
}

const tzng1 = `
#0:record[foo:set[string]]
0:[["test";]]
0:[["testtest";]]`

const tzng2 = `
#0:record[foo:record[bar:string]]
0:[[test;]]`

const tzng3 = `
#0:record[foo:set[string]]
0:[[-;]]`

// String \x2d is "-".
const tzng4 = `
#0:record[foo:bstring]
0:[\x2d;]`

// String \x5b is "[", second string is "[-]" and should pass through.
const tzng5 = `
#0:record[foo:bstring,bar:bstring]
0:[\x5b;\x5b-];]`

// Make sure we handle unset fields and empty sets.
const tzng6 = `
#0:record[id:record[a:string,s:set[string]]]
0:[[-;[]]]`

// Make sure we handle unset sets.
const tzng7 = `
#0:record[a:string,b:set[string],c:set[string]]
0:[foo;[]-;]`

// recursive record with unset set and empty set
const tzng8 = `
#0:record[id:record[a:string,s:set[string]]]
0:[-;]
0:[[-;[]]]
0:[[-;-;]]`

func repeat(c byte, n int) string {
	b := make([]byte, n)
	for k := 0; k < n; k++ {
		b[k] = c
	}
	return string(b)
}

// generate some really big strings
func tzngBig() string {
	s := "#0:record[f0:string,f1:string,f2:string,f3:string]\n"
	s += "0:["
	s += repeat('a', 4) + ";"
	s += repeat('b', 400) + ";"
	s += repeat('c', 30000) + ";"
	s += repeat('d', 2) + ";]\n"
	return s
}

func TestRaw(t *testing.T) {
	boomerang(t, tzng1, false)
	boomerang(t, tzng2, false)
	boomerang(t, tzng3, false)
	boomerang(t, tzng4, false)
	boomerang(t, tzng5, false)
	boomerang(t, tzng6, false)
	boomerang(t, tzng7, false)
	boomerang(t, tzng8, false)
	boomerang(t, tzngBig(), false)
}

func TestRawCompressed(t *testing.T) {
	boomerang(t, tzng1, true)
	boomerang(t, tzng2, true)
	boomerang(t, tzng3, true)
	boomerang(t, tzng4, true)
	boomerang(t, tzng5, true)
	boomerang(t, tzng6, true)
	boomerang(t, tzng7, true)
	boomerang(t, tzng8, true)
	boomerang(t, tzngBig(), true)
}

func TestZjson(t *testing.T) {
	boomerangZJSON(t, tzng1)
	boomerangZJSON(t, tzng2)
	// XXX this one doesn't work right now but it's sort of ok becaue
	// it's a little odd to have an unset string value inside of a set.
	// semantically this would mean the value shouldn't be in the set,
	// but right now this turns into an empty string, which is somewhat reasonable.
	//boomerangZJSON(t, tzng3)
	boomerangZJSON(t, tzng4)
	boomerangZJSON(t, tzng5)
	boomerangZJSON(t, tzng6)
	boomerangZJSON(t, tzng7)
	// XXX need to fix bug in json reader where it always uses a primitive null
	// even within a container type (like json array)
	//boomerangZJSON(t, tzng8)
	boomerangZJSON(t, tzngBig())
}

func TestAlias(t *testing.T) {
	const simple = `#ipaddr=ip
#0:record[foo:string,orig_h:ipaddr]
0:[bar;127.0.0.1;]`

	const wrapped = `#alias1=ip
#alias2=alias1
#alias3=alias2
#0:record[foo:string,orig_h:alias3]
0:[bar;127.0.0.1;]`

	const multipleRecords = `#ipaddr=ip
#0:record[foo:string,orig_h:ipaddr]
0:[bar;127.0.0.1;]
#1:record[foo:string,resp_h:ipaddr]
1:[bro;127.0.0.1;]`
	const recordAlias = `#myrec=record[host:ip]
#0:record[foo:myrec]
0:[[127.0.0.2;]]
0:[-;]`

	t.Run("Zng", func(t *testing.T) {
		t.Run("simple", func(t *testing.T) {
			boomerang(t, simple, true)
		})
		t.Run("wrapped-aliases", func(t *testing.T) {
			boomerang(t, wrapped, true)
		})
		t.Run("alias-in-different-records", func(t *testing.T) {
			boomerang(t, multipleRecords, true)
		})
		t.Run("alias-of-record-type", func(t *testing.T) {
			boomerang(t, recordAlias, true)
		})
	})
	t.Run("ZJSON", func(t *testing.T) {
		t.Run("simple", func(t *testing.T) {
			boomerangZJSON(t, simple)
		})
		t.Run("wrapped-aliases", func(t *testing.T) {
			boomerangZJSON(t, wrapped)
		})
		t.Run("alias-in-different-records", func(t *testing.T) {
			boomerangZJSON(t, multipleRecords)
		})
		t.Run("alias-of-record-type", func(t *testing.T) {
			boomerangZJSON(t, recordAlias)
		})
	})
}

func TestStreams(t *testing.T) {
	const in = `
#0:record[key:ip]
0:[1.2.3.4;]
0:[::;]
0:[1.39.61.22;]
0:[1.149.119.73;]
0:[1.160.203.191;]
0:[2.12.27.251;]`
	tr := tzngio.NewReader(bytes.NewReader([]byte(in)), zson.NewContext())
	var out Output
	zw := zngio.NewWriter(&out, zngio.WriterOpts{
		StreamRecordsMax: 2,
		LZ4BlockSize:     zngio.DefaultLZ4BlockSize,
	})

	var recs []*zng.Record
	for {
		rec, err := tr.Read()
		require.NoError(t, err)
		if rec == nil {
			break
		}
		require.NoError(t, zw.Write(rec))
		recs = append(recs, rec.Keep())
	}

	zr := zngio.NewReader(bytes.NewReader(out.Buffer.Bytes()), zson.NewContext())

	rec, rec2Off, err := zr.SkipStream()
	require.NoError(t, err)
	assert.Equal(t, recs[2].Bytes, rec.Bytes)

	rec, rec4Off, err := zr.SkipStream()
	require.NoError(t, err)
	assert.Equal(t, recs[4].Bytes, rec.Bytes)

	zs := zngio.NewSeeker(bytes.NewReader(out.Buffer.Bytes()), zson.NewContext())

	_, err = zs.Seek(rec4Off)
	require.NoError(t, err)
	rec, err = zs.Read()
	require.NoError(t, err)
	assert.Equal(t, recs[4].Bytes, rec.Bytes)

	_, err = zs.Seek(rec2Off)
	require.NoError(t, err)
	rec, err = zs.Read()
	require.NoError(t, err)
	assert.Equal(t, recs[2].Bytes, rec.Bytes)

	_, err = zs.Seek(0)
	require.NoError(t, err)
	rec, err = zs.Read()
	require.NoError(t, err)
	assert.Equal(t, recs[0].Bytes, rec.Bytes)
}
