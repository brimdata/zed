package tzngio_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/stretchr/testify/assert"
)

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

func identity(t *testing.T, logs string) {
	var out bytes.Buffer
	dst := tzngio.NewWriter(zio.NopCloser(&out))
	in := strings.TrimSpace(logs) + "\n"
	src := tzngio.NewReader(strings.NewReader(in), zed.NewContext())
	err := zio.Copy(dst, src)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.String())
	}
}

// generate some really big strings
func tzngBig() string {
	s := "#0:record[f0:string,f1:string,f2:string,f3:string]\n"
	s += "0:["
	s += strings.Repeat("a", 4) + ";"
	s += strings.Repeat("b", 400) + ";"
	s += strings.Repeat("c", 30000) + ";"
	s += strings.Repeat("d", 2) + ";]\n"
	return s
}
