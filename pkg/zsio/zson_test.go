package zsio

//  This is really a system test dressed up as a unit test.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/zsio/bzson"
	"github.com/mccanne/zq/pkg/zsio/zjson"
	zsonio "github.com/mccanne/zq/pkg/zsio/zson"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
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
	dst := zson.NopFlusher(zsonio.NewWriter(&out))
	in := []byte(strings.TrimSpace(logs) + "\n")
	src := zsonio.NewReader(bytes.NewReader(in), resolver.NewTable())
	err := zson.Copy(dst, src)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

// Send logs to zson reader -> bzson writer -> bzson reader -> zson writer
func boomerang(t *testing.T, logs string) {
	in := []byte(strings.TrimSpace(logs) + "\n")
	zsonSrc := zsonio.NewReader(bytes.NewReader(in), resolver.NewTable())
	var rawZson Output
	rawDst := zson.NopFlusher(bzson.NewWriter(&rawZson))
	err := zson.Copy(rawDst, zsonSrc)
	require.NoError(t, err)

	var out Output
	rawSrc := bzson.NewReader(bytes.NewReader(rawZson.Bytes()), resolver.NewTable())
	zsonDst := zson.NopFlusher(zsonio.NewWriter(&out))
	err = zson.Copy(zsonDst, rawSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
	}
}

func boomerangZJSON(t *testing.T, logs string) {
	zsonSrc := zsonio.NewReader(strings.NewReader(logs), resolver.NewTable())
	var zjsonOutput Output
	zjsonDst := zson.NopFlusher(zjson.NewWriter(&zjsonOutput))
	err := zson.Copy(zjsonDst, zsonSrc)
	require.NoError(t, err)

	var out Output
	zjsonSrc := zjson.NewReader(bytes.NewReader(zjsonOutput.Bytes()), resolver.NewTable())
	zsonDst := zson.NopFlusher(zsonio.NewWriter(&out))
	err = zson.Copy(zsonDst, zjsonSrc)
	if assert.NoError(t, err) {
		assert.Equal(t, strings.TrimSpace(logs), strings.TrimSpace(out.String()))
	}
}

const zson1 = `
#0:record[foo:set[string]]
0:[["test";]]`

const zson2 = `
#0:record[foo:record[bar:string]]
0:[[test;]]`

const zson3 = `
#0:record[foo:set[string]]
0:[[-;]]`

// String \x2d is "-".
const zson4 = `
#0:record[foo:string]
0:[\x2d;]`

// String \x5b is "[", second string is "[-]" and should pass through.
const zson5 = `
#0:record[foo:string,bar:string]
0:[\x5b;\x5b-];]`

// Make sure we handle unset fields and empty sets.
const zson6 = `
#0:record[id:record[a:string,s:set[string]]]
0:[[-;[]]]`

// Make sure we handle unset sets.
const zson7 = `
#0:record[a:string,b:set[string],c:set[string],d:int]
0:[foo;[]-;10;]`

// recursive record with unset set and empty set
const zson8 = `
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
func zsonBig() string {
	s := "#0:record[f0:string,f1:string,f2:string,f3:string]\n"
	s += "0:["
	s += repeat('a', 4) + ";"
	s += repeat('b', 400) + ";"
	s += repeat('c', 30000) + ";"
	s += repeat('d', 2) + ";]\n"
	return s
}

func TestZson(t *testing.T) {
	identity(t, zson1)
	identity(t, zson2)
	identity(t, zson3)
	identity(t, zson4)
	identity(t, zson5)
	identity(t, zson6)
	identity(t, zson7)
	identity(t, zson8)
	identity(t, zsonBig())
}

func TestRaw(t *testing.T) {
	boomerang(t, zson1)
	boomerang(t, zson2)
	boomerang(t, zson3)
	boomerang(t, zson4)
	boomerang(t, zson5)
	boomerang(t, zson6)
	boomerang(t, zson7)
	boomerang(t, zson8)
	boomerang(t, zsonBig())
}

func TestZjson(t *testing.T) {
	boomerangZJSON(t, zson1)
	boomerangZJSON(t, zson2)
	// XXX this one doesn't work right now but it's sort of ok becaue
	// it's a little odd to have an unset string value inside of a set.
	// semantically this would mean the value shouldn't be in the set,
	// but right now this turns into an empty string, which is somewhat reasonable.
	//boomerangZJSON(t, zson3)
	boomerangZJSON(t, zson4)
	boomerangZJSON(t, zson5)
	boomerangZJSON(t, zson6)
	boomerangZJSON(t, zson7)
	boomerangZJSON(t, zson8)
	boomerangZJSON(t, zsonBig())
}

const ctrl = `
#!message1
#0:record[id:record[a:string,s:set[string]]]
#!message2
0:[[-;[]]]
#!message3
#!message4`

func TestCtrl(t *testing.T) {
	// this tests reading of control via text zson,
	// then writing of raw control, and reading back the result
	in := []byte(strings.TrimSpace(ctrl) + "\n")
	r := zsonio.NewReader(bytes.NewReader(in), resolver.NewTable())

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
