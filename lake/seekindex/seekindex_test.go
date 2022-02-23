package seekindex

import (
	"bytes"
	"math"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
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
	s.Lookup(nano.Span{Ts: 100, Dur: 1}, Range{0, 215367}, order.Asc)
	s.Lookup(nano.Span{Ts: 99, Dur: 1}, Range{0, 0}, order.Asc)
	s.Lookup(nano.Span{Ts: 600, Dur: 1}, Range{1139588, 1355498}, order.Asc)
	s.Lookup(nano.Span{Ts: 1000, Dur: 1}, Range{1992947, math.MaxInt64}, order.Asc)
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
	s.Lookup(nano.Span{Ts: 900, Dur: 1}, Range{0, 215367}, order.Desc)
	s.Lookup(nano.Span{Ts: 700, Dur: 1}, Range{438514, 680477}, order.Desc)
	s.Lookup(nano.Span{Ts: 750, Dur: 100}, Range{0, 438514}, order.Desc)
	s.Lookup(nano.Span{Ts: 100, Dur: 1}, Range{1776965, math.MaxInt64}, order.Desc)

}

type entry struct {
	ts     nano.Ts
	offset int64
}

type entries []entry

func (e entries) Order() order.Which {
	if len(e) < 2 || e[0].ts < e[1].ts {
		return order.Asc
	}
	return order.Desc
}

type testSeekIndex struct {
	*testing.T
	buffer *bytes.Buffer
}

func (t *testSeekIndex) Lookup(s nano.Span, expected Range, o order.Which) {
	r := zngio.NewReader(bytes.NewReader(t.buffer.Bytes()), zed.NewContext())
	defer r.Close()
	cmp := extent.CompareFunc(o)
	var first, last *zed.Value
	if o == order.Asc {
		first = zed.NewTime(s.Ts)
		last = zed.NewTime(s.End() - 1)
	} else {
		first = zed.NewTime(s.End() - 1)
		last = zed.NewTime(s.Ts)
	}
	rg, err := Lookup(r, first, last, cmp)
	require.NoError(t, err)
	assert.Equal(t, expected, rg)
}

func newTestSeekIndex(t *testing.T, entries []entry) *testSeekIndex {
	b := build(t, entries)
	return &testSeekIndex{T: t, buffer: b}
}

func build(t *testing.T, entries entries) *bytes.Buffer {
	var buffer bytes.Buffer
	w := NewWriter(zngio.NewWriter(zio.NopCloser(&buffer), zngio.WriterOpts{}))
	for i, entry := range entries {
		zv := zed.Value{zed.TypeTime, zed.EncodeTime(entry.ts)}
		err := w.Write(zv, uint64(i), entry.offset)
		require.NoError(t, err)
	}
	return &buffer
}
