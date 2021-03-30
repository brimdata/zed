package tzngio_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng/resolver"
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
	r := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())

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
		zngSrc := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())
		zngDst := tzngio.NewWriter(&output{})
		err := zbuf.Copy(zngDst, zngSrc)
		assert.Errorf(t, err, errorMsg, errorArgs...)
	})
}

// Send logs to zng reader ->  zng writer
func boomerang(t *testing.T, name, logs string) {
	t.Run(name, func(t *testing.T) {
		var out output
		in := []byte(strings.TrimSpace(logs) + "\n")
		zngSrc := tzngio.NewReader(bytes.NewReader(in), resolver.NewContext())
		zngDst := tzngio.NewWriter(&out)
		err := zbuf.Copy(zngDst, zngSrc)
		require.NoError(t, err)
		assert.Equal(t, string(in), out.String())
	})
}
