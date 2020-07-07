package driver

import (
	"context"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
)

type counter struct {
	n int
}

func (c *counter) Write(*zng.Record) error {
	c.n++
	return nil
}

func TestMuxDriver(t *testing.T) {
	input := `
#0:record[_path:string,ts:time]
0:[conn;1425565514.419939;]`

	zctx := resolver.NewContext()
	query, err := zql.ParseProc("(tail 1; tail 1)")
	assert.NoError(t, err)

	t.Run("muxed into one writer", func(t *testing.T) {
		reader := tzngio.NewReader(strings.NewReader(input), zctx)
		flowgraph, err := Compile(context.Background(), resolver.NewContext(), query, reader, "", false, nano.MaxSpan, nil)
		assert.NoError(t, err)
		c := counter{}
		d := NewCLI(&c)
		err = Run(flowgraph, d, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, c.n)
	})

	t.Run("muxed into individual writers", func(t *testing.T) {
		reader := tzngio.NewReader(strings.NewReader(input), zctx)
		flowgraph, err := Compile(context.Background(), resolver.NewContext(), query, reader, "", false, nano.MaxSpan, nil)
		assert.NoError(t, err)
		cs := []zbuf.Writer{&counter{}, &counter{}}
		d := NewCLI(cs...)
		err = Run(flowgraph, d, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, cs[0].(*counter).n)
		assert.Equal(t, 1, cs[1].(*counter).n)
	})
}
