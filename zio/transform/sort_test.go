package transform_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/transform"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

func TestAscending(t *testing.T) {
	const input = `
#0:record[ts:time,value:int32]
0:[6;1;]
0:[5;2;]
0:[4;3;]
0:[3;4;]
0:[2;5;]
0:[1;6;]`

	const expected = `
#0:record[ts:time,value:int32]
0:[1;6;]
0:[2;5;]
0:[3;4;]
0:[4;3;]
0:[5;2;]
0:[6;1;]`
	t.Run("InMemory", func(t *testing.T) {
		r := tzngio.NewReader(strings.NewReader(input), resolver.NewContext())
		sr := transform.NewSortReader(context.Background(), r, 0, expr.SortTsAscending, "")
		runTest(t, sr, expected)
	})
	t.Run("SpillToDisk", func(t *testing.T) {
		r := tzngio.NewReader(strings.NewReader(input), resolver.NewContext())
		sr := transform.NewSortReader(context.Background(), r, 1, expr.SortTsAscending, "")
		runTest(t, sr, expected)
	})
}

func TestDescending(t *testing.T) {
	const input = `
#0:record[ts:time,value:int32]
0:[1;1;]
0:[2;2;]
0:[3;3;]
0:[4;4;]
0:[5;5;]
0:[6;6;]
`
	const expected = `
#0:record[ts:time,value:int32]
0:[6;6;]
0:[5;5;]
0:[4;4;]
0:[3;3;]
0:[2;2;]
0:[1;1;]`
	t.Run("InMemory", func(t *testing.T) {
		r := tzngio.NewReader(strings.NewReader(input), resolver.NewContext())
		sr := transform.NewSortReader(context.Background(), r, 0, expr.SortTsDescending, "")
		runTest(t, sr, expected)
	})
	t.Run("SpillToDisk", func(t *testing.T) {
		r := tzngio.NewReader(strings.NewReader(input), resolver.NewContext())
		sr := transform.NewSortReader(context.Background(), r, 1, expr.SortTsDescending, "")
		runTest(t, sr, expected)
	})
}

func runTest(t *testing.T, sr *transform.SortReader, expected string) {
	buf := bytes.NewBuffer(nil)
	err := zbuf.Copy(zbuf.NopFlusher(tzngio.NewWriter(buf)), sr)
	require.NoError(t, err)
	require.Equal(t, test.Trim(expected), buf.String())
}
