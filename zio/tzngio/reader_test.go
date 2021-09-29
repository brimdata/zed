package tzngio_test

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	r := tzngio.NewReader(bytes.NewReader(in), zed.NewContext())

	_, body, err := r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, "message1", string(body))

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, "message2", string(body))

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.True(t, body == nil)

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, "message3", string(body))

	_, body, err = r.ReadPayload()
	assert.NoError(t, err)
	assert.Equal(t, "message4", string(body))
}

func TestDescriptors(t *testing.T) {
	// Step 1 - Test a simple zng descriptor and corresponding value
	src := "#port=uint16\n"
	src += "#1:record[s:string,n:int32]\n"
	src += "1:[foo;5;]\n"
	// Step 2 - Create a second descriptor of a different type
	src += "#2:record[a:ip,p:port]\n"
	src += "2:[10.5.5.5;443;]\n"
	// Step 3 - can still use the first descriptor
	src += "1:[bar;100;]\n"
	// Step 4 - Test that referencing an invalid descriptor is an error.
	src += "100:[something;somethingelse;]\n"

	r := tzngio.NewReader(strings.NewReader(src), zed.NewContext())

	// Check Step 1
	record, err := r.Read()
	require.NoError(t, err)
	s, err := record.AccessString("s")
	require.NoError(t, err)
	assert.Equal(t, "foo", s, "Parsed string value properly")
	n, err := record.AccessInt("n")
	require.NoError(t, err)
	assert.Equal(t, 5, int(n), "Parsed int value properly")

	// Check Step 2
	record, err = r.Read()
	require.NoError(t, err)
	a, err := record.AccessIP("a")
	require.NoError(t, err)
	expectAddr := net.ParseIP("10.5.5.5").To4()
	assert.Equal(t, expectAddr, a, "Parsed addr value properly")
	n, err = record.AccessInt("p")
	require.NoError(t, err)
	assert.Equal(t, 443, int(n), "Parsed port value properly")

	// Check Step 3
	record, err = r.Read()
	require.NoError(t, err)
	s, err = record.AccessString("s")
	require.NoError(t, err)
	assert.Equal(t, "bar", s, "Parsed another string properly")
	n, err = record.AccessInt("n")
	require.NoError(t, err)
	assert.Equal(t, 100, int(n), "Parsed another int properly")

	// XXX test other types, sets, arrays, etc.

	// Check Step 4 - Test that referencing an invalid descriptor is an error.
	_, err = r.Read()
	assert.Error(t, err, "invalid descriptor", "invalid descriptor")

	// Test various malformed zng:
	def1 := "#1:record[s:string,n:int32]\n"
	zngs := []string{
		def1 + "1:string;123;\n",  // missing brackets
		def1 + "1:[string;123]\n", // missing semicolon
	}

	for _, z := range zngs {
		r := tzngio.NewReader(strings.NewReader(z), zed.NewContext())
		_, err = r.Read()
		assert.Error(t, err, "tzng parse error", "invalid tzng")
	}

	// Descriptor with an invalid type is rejected
	r = tzngio.NewReader(strings.NewReader("#4:notatype\n"), zed.NewContext())
	_, err = r.Read()
	assert.Error(t, err, "unknown type", "descriptor with invalid type")
}

func TestSyntax(t *testing.T) {
	const bad1 = `
#0:record[_path:string,ts:time,uid:string,resp_ip_bytes:count,tunnel_parents:set[string]]
0:[conn;1425565514.419939;CogZFI3py5JsFZGik;0;]`
	r := tzngio.NewReader(strings.NewReader(bad1), zed.NewContext())
	_, err := r.Read()
	require.Error(t, err, "bad1 must have error")

	const bad2 = `
#0:record[a:string,record[b:string]]
0:[foo;[bar;]]`
	r = tzngio.NewReader(strings.NewReader(bad2), zed.NewContext())
	_, err = r.Read()
	require.Error(t, err, "bad2 must have error")

	const bad3 = `
#0:record[_path:string,ts:time,uid:string,resp_ip_bytes:count,tunnel_parents:set[string]]
0:[conn;1425565514.419939;CogZFI3py5JsFZGik;0;0;[]]`
	r = tzngio.NewReader(strings.NewReader(bad3), zed.NewContext())
	_, err = r.Read()
	require.Error(t, err, "bad3 must have error")

}

func TestAlias(t *testing.T) {
	boomerang(t, "simple", `
#ipaddr=ip
#0:record[foo:string,orig_h:ipaddr]
0:[bar;127.0.0.1;]`)

	boomerang(t, "wrapped-aliases", `
#alias1=ip
#alias2=alias1
#alias3=alias2
#0:record[foo:string,orig_h:alias3]
0:[bar;127.0.0.1;]`)

	boomerang(t, "alias-in-different-records", `
#ipaddr=ip
#0:record[foo:string,orig_h:ipaddr]
0:[bar;127.0.0.1;]
#1:record[foo:string,resp_h:ipaddr]
1:[bro;127.0.0.1;]`)

	boomerang(t, "same-primitive-different-records", `
#ipaddr=ip
#0:record[foo:string,orig_h:ipaddr]
0:[bro;127.0.0.1;]
#1:record[foo:string,orig_h:ip]
1:[bar;127.0.0.1;]`)

	boomerang(t, "alias to record", `
#rec=record[s:string]
#0:record[r:rec]
0:[[hello;]]`)

	boomerang(t, "alias to array", `
#arr=array[int32]
#0:record[a:arr]
0:[[1;2;3;]]`)

	boomerang(t, "alias to set", `
#ss=set[string]
#0:record[s:ss]
0:[[a;b;c;]]`)
}

func TestAliasErr(t *testing.T) {
	boomerangErr(t, "non-existent", `
#ip=doesnotexist
#0:record[foo:string,orig_h:ip]`, "unknown type: %s", "doesnotexist")

	boomerangErr(t, "out-of-order", `
#alias3=alias2
#alias2=alias1
#alias1=ip
#0:record[foo:string,orig_h:alias3]`, "unknown type: %s", "alias2")

	boomerangErr(t, "alias-preexisting", `
#interval=alias
#0:record[foo:string,dur:interval]`, "unknown type: %s", "alias")
}

type output struct {
	bytes.Buffer
}

func (o *output) Close() error { return nil }

func boomerangErr(t *testing.T, name, logs, errorMsg string, errorArgs ...interface{}) {
	t.Run(name, func(t *testing.T) {
		in := []byte(strings.TrimSpace(logs) + "\n")
		zngSrc := tzngio.NewReader(bytes.NewReader(in), zed.NewContext())
		zngDst := tzngio.NewWriter(&output{})
		err := zio.Copy(zngDst, zngSrc)
		assert.Errorf(t, err, errorMsg, errorArgs...)
	})
}

// Send logs to zng reader ->  zng writer
func boomerang(t *testing.T, name, logs string) {
	t.Run(name, func(t *testing.T) {
		var out output
		in := []byte(strings.TrimSpace(logs) + "\n")
		zngSrc := tzngio.NewReader(bytes.NewReader(in), zed.NewContext())
		zngDst := tzngio.NewWriter(&out)
		err := zio.Copy(zngDst, zngSrc)
		require.NoError(t, err)
		assert.Equal(t, string(in), out.String())
	})
}
