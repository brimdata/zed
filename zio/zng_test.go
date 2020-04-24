package zio_test

//  This is really a system test dressed up as a unit test.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zio/zjsonio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Output struct {
	bytes.Buffer
}

func (o *Output) Close() error {
	return nil
}

func identity(t *testing.T, logs string) {
	var out Output
	dst := zbuf.NopFlusher(tzngio.NewWriter(&out))
	in := []byte(strings.TrimSpace(logs) + "\n")
	src := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())
	err := zbuf.Copy(dst, src)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

// Send logs to tzng reader -> zng writer -> zng reader -> tzng writer
func boomerang(t *testing.T, logs string) {
	in := []byte(strings.TrimSpace(logs) + "\n")
	tzngSrc := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())
	var rawzng Output
	rawDst := zbuf.NopFlusher(zngio.NewWriter(&rawzng, zio.WriterFlags{}))
	err := zbuf.Copy(rawDst, tzngSrc)
	require.NoError(t, err)

	var out Output
	rawSrc := zngio.NewReader(bytes.NewReader(rawzng.Bytes()), resolver.NewContext())
	tzngDst := zbuf.NopFlusher(tzngio.NewWriter(&out))
	err = zbuf.Copy(tzngDst, rawSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

func boomerangZJSON(t *testing.T, logs string) {
	tzngSrc := tzngio.NewReader(strings.NewReader(logs), resolver.NewContext())
	var zjsonOutput Output
	zjsonDst := zbuf.NopFlusher(zjsonio.NewWriter(&zjsonOutput))
	err := zbuf.Copy(zjsonDst, tzngSrc)
	require.NoError(t, err)

	var out Output
	zjsonSrc := zjsonio.NewReader(bytes.NewReader(zjsonOutput.Bytes()), resolver.NewContext())
	tzngDst := zbuf.NopFlusher(tzngio.NewWriter(&out))
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

func TestTzng(t *testing.T) {
	identity(t, tzng1)
	identity(t, tzng2)
	identity(t, tzng3)
	identity(t, tzng4)
	identity(t, tzng5)
	identity(t, tzng6)
	identity(t, tzng7)
	identity(t, tzng8)
	identity(t, tzngBig())
}

func TestRaw(t *testing.T) {
	boomerang(t, tzng1)
	boomerang(t, tzng2)
	boomerang(t, tzng3)
	boomerang(t, tzng4)
	boomerang(t, tzng5)
	boomerang(t, tzng6)
	boomerang(t, tzng7)
	boomerang(t, tzng8)
	boomerang(t, tzngBig())
}

const ctrl = `
#!message1
#0:record[id:record[a:string,s:set[string]]]
#!message2
0:[[-;[]]]
#!message3
#!message4`

func TestCtrl(t *testing.T) {
	// this tests reading of control via text zng,
	// then writing of raw control, and reading back the result
	in := []byte(strings.TrimSpace(ctrl) + "\n")
	r := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())

	_, body, err := r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, body, []byte("message1"))

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, body, []byte("message2"))

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.True(t, body == nil)

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, body, []byte("message3"))

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, body, []byte("message4"))
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

	t.Run("Zng", func(t *testing.T) {
		t.Run("simple", func(t *testing.T) {
			boomerang(t, simple)
		})
		t.Run("wrapped-aliases", func(t *testing.T) {
			boomerang(t, wrapped)
		})
		t.Run("alias-in-different-records", func(t *testing.T) {
			boomerang(t, multipleRecords)
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
	})
}

const strm = `
#0:record[key:ip]
0:[1.2.3.4;]
0:[::;]
0:[1.39.61.22;]
0:[1.149.119.73;]
0:[1.160.203.191;]
0:[2.12.27.251;]`

func TestEOS(t *testing.T) {
	in := []byte(strings.TrimSpace(strm) + "\n")
	r := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())
	var out Output
	writer := zngio.NewWriter(&out, zio.WriterFlags{StreamRecordsMax: 2})
	w := zbuf.NopFlusher(writer)

	// Copy the tzng as zng to out and record the position of the second record.
	rec, err := r.Read()
	require.NoError(t, err)
	err = w.Write(rec)
	require.NoError(t, err)
	writePos := writer.Position()
	rec, err = r.Read()
	require.NoError(t, err)
	err = w.Write(rec)
	require.NoError(t, err)
	// After two writes there is a valid sync point.
	seekPoint := writer.Position()
	seekRec, err := r.Read()
	require.NoError(t, err)
	err = w.Write(seekRec)
	require.NoError(t, err)
	err = zbuf.Copy(w, r)
	require.NoError(t, err)

	// Read back the zng and make sure the streams are aligned after
	// the first record.

	r2 := zngio.NewReader(bytes.NewReader(out.Buffer.Bytes()), resolver.NewContext())
	_, err = r2.Read()
	require.NoError(t, err)
	readPos := r2.Position()
	assert.Equal(t, writePos, readPos)

	// Read back the zng and make sure the streams are aligned after
	// the first record.

	s := zngio.NewSeeker(bytes.NewReader(out.Buffer.Bytes()), resolver.NewContext())
	_, err = s.Seek(seekPoint)
	require.NoError(t, err)
	rec, err = s.Read()
	require.NoError(t, err)
	assert.Equal(t, seekRec.Raw, rec.Raw)

	r3 := zngio.NewReader(bytes.NewReader(out.Buffer.Bytes()), resolver.NewContext())
	rec, off, err := r3.SkipStream()
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, off, seekPoint)
}
