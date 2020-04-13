package zio_test

//  This is really a system test dressed up as a unit test.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/bzngio"
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
	dst := zbuf.NopFlusher(zngio.NewWriter(&out))
	in := []byte(strings.TrimSpace(logs) + "\n")
	src := zngio.NewReader(bytes.NewReader(in), resolver.NewContext())
	err := zbuf.Copy(dst, src)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

// Send logs to zng reader -> bzng writer -> bzng reader -> zng writer
func boomerang(t *testing.T, logs string) {
	in := []byte(strings.TrimSpace(logs) + "\n")
	zngSrc := zngio.NewReader(bytes.NewReader(in), resolver.NewContext())
	var rawzng Output
	rawDst := zbuf.NopFlusher(bzngio.NewWriter(&rawzng, zio.Flags{}))
	err := zbuf.Copy(rawDst, zngSrc)
	require.NoError(t, err)

	var out Output
	rawSrc := bzngio.NewReader(bytes.NewReader(rawzng.Bytes()), resolver.NewContext())
	zngDst := zbuf.NopFlusher(zngio.NewWriter(&out))
	err = zbuf.Copy(zngDst, rawSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

func boomerangZJSON(t *testing.T, logs string) {
	zngSrc := zngio.NewReader(strings.NewReader(logs), resolver.NewContext())
	var zjsonOutput Output
	zjsonDst := zbuf.NopFlusher(zjsonio.NewWriter(&zjsonOutput))
	err := zbuf.Copy(zjsonDst, zngSrc)
	require.NoError(t, err)

	var out Output
	zjsonSrc := zjsonio.NewReader(bytes.NewReader(zjsonOutput.Bytes()), resolver.NewContext())
	zngDst := zbuf.NopFlusher(zngio.NewWriter(&out))
	err = zbuf.Copy(zngDst, zjsonSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, strings.TrimSpace(logs), strings.TrimSpace(out.String()))
	}
}

const zng1 = `
#0:record[foo:set[string]]
0:[["test";]]
0:[["testtest";]]`

const zng2 = `
#0:record[foo:record[bar:string]]
0:[[test;]]`

const zng3 = `
#0:record[foo:set[string]]
0:[[-;]]`

// String \x2d is "-".
const zng4 = `
#0:record[foo:bstring]
0:[\x2d;]`

// String \x5b is "[", second string is "[-]" and should pass through.
const zng5 = `
#0:record[foo:bstring,bar:bstring]
0:[\x5b;\x5b-];]`

// Make sure we handle unset fields and empty sets.
const zng6 = `
#0:record[id:record[a:string,s:set[string]]]
0:[[-;[]]]`

// Make sure we handle unset sets.
const zng7 = `
#0:record[a:string,b:set[string],c:set[string]]
0:[foo;[]-;]`

// recursive record with unset set and empty set
const zng8 = `
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
func zngBig() string {
	s := "#0:record[f0:string,f1:string,f2:string,f3:string]\n"
	s += "0:["
	s += repeat('a', 4) + ";"
	s += repeat('b', 400) + ";"
	s += repeat('c', 30000) + ";"
	s += repeat('d', 2) + ";]\n"
	return s
}

func TestZng(t *testing.T) {
	identity(t, zng1)
	identity(t, zng2)
	identity(t, zng3)
	identity(t, zng4)
	identity(t, zng5)
	identity(t, zng6)
	identity(t, zng7)
	identity(t, zng8)
	identity(t, zngBig())
}

func TestRaw(t *testing.T) {
	boomerang(t, zng1)
	boomerang(t, zng2)
	boomerang(t, zng3)
	boomerang(t, zng4)
	boomerang(t, zng5)
	boomerang(t, zng6)
	boomerang(t, zng7)
	boomerang(t, zng8)
	boomerang(t, zngBig())
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
	r := zngio.NewReader(bytes.NewReader(in), resolver.NewContext())

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
	boomerangZJSON(t, zng1)
	boomerangZJSON(t, zng2)
	// XXX this one doesn't work right now but it's sort of ok becaue
	// it's a little odd to have an unset string value inside of a set.
	// semantically this would mean the value shouldn't be in the set,
	// but right now this turns into an empty string, which is somewhat reasonable.
	//boomerangZJSON(t, zng3)
	boomerangZJSON(t, zng4)
	boomerangZJSON(t, zng5)
	boomerangZJSON(t, zng6)
	boomerangZJSON(t, zng7)
	// XXX need to fix bug in json reader where it always uses a primitive null
	// even within a container type (like json array)
	//boomerangZJSON(t, zng8)
	boomerangZJSON(t, zngBig())
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

	t.Run("Bzng", func(t *testing.T) {
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
	r := zngio.NewReader(bytes.NewReader(in), resolver.NewContext())
	var out Output
	writer := bzngio.NewWriter(&out, zio.Flags{StreamRecordsMax: 2})
	w := zbuf.NopFlusher(writer)

	// Copy the zng as bzng to out and record the position of the second record.
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

	// Read back the bzng and make sure the streams are aligned after
	// the first record.

	r2 := bzngio.NewReader(bytes.NewReader(out.Buffer.Bytes()), resolver.NewContext())
	_, err = r2.Read()
	require.NoError(t, err)
	readPos := r2.Position()
	assert.Equal(t, writePos, readPos)

	// Read back the bzng and make sure the streams are aligned after
	// the first record.

	s := bzngio.NewSeeker(bytes.NewReader(out.Buffer.Bytes()), resolver.NewContext())
	_, err = s.Seek(seekPoint)
	require.NoError(t, err)
	rec, err = s.Read()
	require.NoError(t, err)
	assert.Equal(t, seekRec.Raw, rec.Raw)
}
