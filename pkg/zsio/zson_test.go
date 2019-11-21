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
0:[["test";];];`

const zson2 = `
#0:record[foo:record[bar:string]]
0:[[test;];];`

const zson3 = `
#0:record[foo:set[string]]
0:[[-;];];`

func TestZson(t *testing.T) {
	identity(t, zson1)
	identity(t, zson2)
	identity(t, zson3)
}
