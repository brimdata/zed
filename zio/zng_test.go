package zio_test

//  This is really a system test dressed up as a unit test.

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Output struct {
	bytes.Buffer
}

func (o *Output) Close() error {
	return nil
}

// Send logs to ZSON reader -> ZNG writer -> ZNG reader -> ZSON writer.
func boomerang(t *testing.T, logs string, compress bool) {
	in := []byte(strings.TrimSpace(logs) + "\n")
	zsonSrc := zsonio.NewReader(bytes.NewReader(in), zed.NewContext())
	var rawzng Output
	var zngLZ4BlockSize int
	if compress {
		zngLZ4BlockSize = zngio.DefaultLZ4BlockSize
	}
	rawDst := zngio.NewWriter(&rawzng, zngio.WriterOpts{LZ4BlockSize: zngLZ4BlockSize})
	require.NoError(t, zio.Copy(rawDst, zsonSrc))
	require.NoError(t, rawDst.Close())

	var out Output
	rawSrc := zngio.NewReader(bytes.NewReader(rawzng.Bytes()), zed.NewContext())
	zsonDst := zsonio.NewWriter(&out, zsonio.WriterOpts{})
	err := zio.Copy(zsonDst, rawSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

func boomerangZJSON(t *testing.T, logs string) {
	zsonSrc := zsonio.NewReader(strings.NewReader(logs), zed.NewContext())
	var zjsonOutput Output
	zjsonDst := zjsonio.NewWriter(&zjsonOutput)
	err := zio.Copy(zjsonDst, zsonSrc)
	require.NoError(t, err)

	var out Output
	zjsonSrc := zjsonio.NewReader(bytes.NewReader(zjsonOutput.Bytes()), zed.NewContext())
	zsonDst := zsonio.NewWriter(&out, zsonio.WriterOpts{})
	err = zio.Copy(zsonDst, zjsonSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, strings.TrimSpace(logs), strings.TrimSpace(out.String()))
	}
}

const zson1 = `
{foo:|["\"test\""]|}
{foo:|["\"testtest\""]|}
`

const zson2 = `{foo:{bar:"test"}}`

const zson3 = "{foo:|[null(string)]|}"

const zson4 = `{foo:"-"}`

const zson5 = `{foo:"[",bar:"[-]"}`

// Make sure we handle null fields and empty sets.
const zson6 = "{id:{a:null(string),s:|[]|(|[string]|)}}"

// Make sure we handle empty and null sets.
const zson7 = `{a:"foo",b:|[]|(|[string]|),c:null(|[string]|)}`

// recursive record with null set and empty set
const zson8 = `
{id:{a:null(string),s:|[]|(|[string]|)}}
{id:{a:null(string),s:null(|[string]|)}}
{id:null({a:string,s:|[string]|})}
`

// generate some really big strings
func zsonBig() string {
	return fmt.Sprintf(`{f0:"%s",f1:"%s",f2:"%s",f3:"%s"}`,
		"aaaa", strings.Repeat("b", 400), strings.Repeat("c", 30000), "dd")
}

func TestRaw(t *testing.T) {
	boomerang(t, zson1, false)
	boomerang(t, zson2, false)
	boomerang(t, zson3, false)
	boomerang(t, zson4, false)
	boomerang(t, zson5, false)
	boomerang(t, zson6, false)
	boomerang(t, zson7, false)
	boomerang(t, zson8, false)
	boomerang(t, zsonBig(), false)
}

func TestRawCompressed(t *testing.T) {
	boomerang(t, zson1, true)
	boomerang(t, zson2, true)
	boomerang(t, zson3, true)
	boomerang(t, zson4, true)
	boomerang(t, zson5, true)
	boomerang(t, zson6, true)
	boomerang(t, zson7, true)
	boomerang(t, zson8, true)
	boomerang(t, zsonBig(), true)
}

func TestZjson(t *testing.T) {
	boomerangZJSON(t, zson1)
	boomerangZJSON(t, zson2)
	// XXX this one doesn't work right now but it's sort of ok becaue
	// it's a little odd to have an null string value inside of a set.
	// semantically this would mean the value shouldn't be in the set,
	// but right now this turns into an empty string, which is somewhat reasonable.
	//boomerangZJSON(t, zson3)
	boomerangZJSON(t, zson4)
	boomerangZJSON(t, zson5)
	boomerangZJSON(t, zson6)
	boomerangZJSON(t, zson7)
	// XXX need to fix bug in json reader where it always uses a primitive null
	// even within a container type (like json array)
	//boomerangZJSON(t, zson8)
	boomerangZJSON(t, zsonBig())
}

func TestAlias(t *testing.T) {
	const simple = `{foo:"bar",orig_h:127.0.0.1(=ipaddr)}`
	const multipleRecords = `
{foo:"bar",orig_h:127.0.0.1(=ipaddr)}
{foo:"bro",resp_h:127.0.0.1(=ipaddr)}
`
	const recordAlias = `
{foo:{host:127.0.0.2}(=myrec)}
{foo:null(myrec=<{host:ip}>)}
`
	t.Run("ZNG", func(t *testing.T) {
		t.Run("simple", func(t *testing.T) {
			boomerang(t, simple, true)
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
{key:1.2.3.4}
{key:::}
{key:1.39.61.22}
{key:1.149.119.73}
{key:1.160.203.191}
{key:2.12.27.251}
`
	r := zsonio.NewReader(strings.NewReader(in), zed.NewContext())
	var out Output
	zw := zngio.NewWriter(&out, zngio.WriterOpts{
		LZ4BlockSize: zngio.DefaultLZ4BlockSize,
	})

	var recs []*zed.Value
	for {
		rec, err := r.Read()
		require.NoError(t, err)
		if rec == nil {
			break
		}
		require.NoError(t, zw.Write(rec))
		recs = append(recs, rec.Copy())
		if len(recs)%2 == 0 {
			require.NoError(t, zw.EndStream())
		}
	}

	zr := zngio.NewReader(bytes.NewReader(out.Buffer.Bytes()), zed.NewContext())

	rec, rec2Off, err := zr.SkipStream()
	require.NoError(t, err)
	assert.Equal(t, recs[2].Bytes, rec.Bytes)

	rec, rec4Off, err := zr.SkipStream()
	require.NoError(t, err)
	assert.Equal(t, recs[4].Bytes, rec.Bytes)

	b := out.Buffer.Bytes()
	len := int64(len(b))

	sr := io.NewSectionReader(bytes.NewReader(b), rec4Off, len-rec4Off)
	reader := zngio.NewReader(sr, zed.NewContext())
	rec, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, recs[4].Bytes, rec.Bytes)

	sr = io.NewSectionReader(bytes.NewReader(b), rec2Off, len-rec2Off)
	reader = zngio.NewReader(sr, zed.NewContext())
	rec, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, recs[2].Bytes, rec.Bytes)

	sr = io.NewSectionReader(bytes.NewReader(b), 0, len)
	reader = zngio.NewReader(sr, zed.NewContext())
	rec, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, recs[0].Bytes, rec.Bytes)
}
