package zsio

//  This is really a system test dressed up as a unit test.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/stretchr/testify/assert"
)

type Output struct {
	bytes.Buffer
}

func (o *Output) Close() error {
	return nil
}

func identity(t *testing.T, logs string) {
	var out Output
	dst := NewWriter(&out)
	in := []byte(strings.TrimSpace(logs) + "\n")
	src := NewReader(bytes.NewReader(in), resolver.NewTable())
	err := zson.Copy(dst, src)
	if assert.NoError(t, err) {
		assert.Equal(t, in, out.Bytes())
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

// string \x2d is "-"
const zson4 = `
#0:record[foo:string]
0:[\x2d;]`

// string \x5b is "[", second string is "[-]" and should pass through
const zson5 = `
#0:record[foo:string,bar:string]
0:[\x5b;\x5b-];]`

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
	identity(t, zsonBig())
}
