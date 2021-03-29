package seekindex

import (
	"context"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAscending(t *testing.T) {
	var entries = []entry{
		{100, 0},
		{200, 215367},
		{300, 438514},
		{400, 680477},
		{500, 904528},
		{600, 1139588},
		{700, 1355498},
		{800, 1564211},
		{900, 1776965},
		{1000, 1992947},
	}
	s := newTestSeekIndex(t, entries)
	s.Lookup(nano.Span{Ts: 100, Dur: 1}, Range{0, 215367})
	s.Lookup(nano.Span{Ts: 99, Dur: 1}, Range{0, 0})
	s.Lookup(nano.Span{Ts: 600, Dur: 1}, Range{1139588, 1355498})
	s.Lookup(nano.Span{Ts: 1000, Dur: 1}, Range{1992947, math.MaxInt64})
}

func TestDescending(t *testing.T) {
	var entries = []entry{
		{900, 0},
		{800, 215367},
		{700, 438514},
		{600, 680477},
		{500, 904528},
		{400, 1139588},
		{300, 1355498},
		{200, 1564211},
		{100, 1776965},
	}
	s := newTestSeekIndex(t, entries)
	s.Lookup(nano.Span{Ts: 900, Dur: 1}, Range{0, 215367})
	s.Lookup(nano.Span{Ts: 700, Dur: 1}, Range{438514, 680477})
	s.Lookup(nano.Span{Ts: 750, Dur: 100}, Range{0, 438514})
	s.Lookup(nano.Span{Ts: 100, Dur: 1}, Range{1776965, math.MaxInt64})

}

type entry struct {
	ts     nano.Ts
	offset int64
}

type entries []entry

func (e entries) Order() zbuf.Order {
	if len(e) < 2 || e[0].ts < e[1].ts {
		return zbuf.OrderAsc
	}
	return zbuf.OrderDesc
}

type testSeekIndex struct {
	*SeekIndex
	*testing.T
}

func (t *testSeekIndex) Lookup(span nano.Span, expected Range) {
	rg, err := t.SeekIndex.Lookup(context.Background(), span)
	require.NoError(t, err)
	assert.Equal(t, expected, rg)
}

func newTestSeekIndex(t *testing.T, entries []entry) *testSeekIndex {
	path := build(t, entries)
	s, err := Open(context.Background(), iosrc.MustParseURI(path))
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	require.NoError(t, err)
	return &testSeekIndex{s, t}
}

func build(t *testing.T, entries entries) string {
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(dir) })
	path := filepath.Join(dir, "seekindex.zng")
	builder, err := NewBuilder(context.Background(), path, entries.Order())
	require.NoError(t, err)
	for _, entry := range entries {
		err = builder.Enter(entry.ts, entry.offset)
		require.NoError(t, err)
	}
	require.NoError(t, builder.Close())
	return path
}
