package zngio_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zio/zngio"
	"github.com/mccanne/zq/zng/resolver"
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

func TestAlias(t *testing.T) {
	boomerang(t, "simple", `
#ip=addr
#0:record[foo:string,orig_h:ip]
0:[bar;127.0.0.1;]`)

	boomerang(t, "wrapped-aliases", `
#alias1=addr
#alias2=alias1
#alias3=alias2
#0:record[foo:string,orig_h:alias3]
0:[bar;127.0.0.1;]`)

	boomerang(t, "alias-in-different-records", `
#ip=addr
#0:record[foo:string,orig_h:ip]
0:[bar;127.0.0.1;]
#1:record[foo:string,resp_h:ip]
1:[bro;127.0.0.1;]`)

	boomerang(t, "same-primitive-different-records", `
#ip=addr
#0:record[foo:string,orig_h:ip]
0:[bro;127.0.0.1;]
#1:record[foo:string,orig_h:addr]
1:[bar;127.0.0.1;]`)

	boomerang(t, "redefine-alias", `
#alias=addr
#0:record[orig_h:alias]
0:[127.0.0.1;]
#alias=count
#1:record[count:alias]
1:[25;]`)
}

func TestAliasErr(t *testing.T) {
	boomerangErr(t, "non-existent", `
#ip=doesnotexist
#0:record[foo:string,orig_h:ip]`, "unknown type: %s", "doesnotexist")

	boomerangErr(t, "out-of-order", `
#alias3=alias2
#alias2=alias1
#alias1=addr
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
		zngSrc := zngio.NewReader(bytes.NewReader(in), resolver.NewContext())
		zngDst := zbuf.NopFlusher(zngio.NewWriter(&output{}))
		err := zbuf.Copy(zngDst, zngSrc)
		assert.Errorf(t, err, errorMsg, errorArgs...)
	})
}

// Send logs to zng reader ->  zng writer
func boomerang(t *testing.T, name, logs string) {
	t.Run(name, func(t *testing.T) {
		var out output
		in := []byte(strings.TrimSpace(logs) + "\n")
		zngSrc := zngio.NewReader(bytes.NewReader(in), resolver.NewContext())
		zngDst := zbuf.NopFlusher(zngio.NewWriter(&out))
		err := zbuf.Copy(zngDst, zngSrc)
		require.NoError(t, err)
		assert.Equal(t, string(in), out.String())
	})
}
