package zval

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var appendCases = [][][]byte{
	{},
	{nil},
	{[]byte{}},
	{[]byte{}, nil},
	{[]byte("data")},
	{[]byte("\x00\x01\x02")},
	{[]byte("UTF-8 \b5Ὂg̀9!℃ᾭG€�")},
	{[]byte("data"), nil, []byte("\x1a\x2b\x3c"), []byte("UTF-8 \b5Ὂg̀9!℃ᾭG€�")},
	{[]byte("thisisareallylongstringdoyoulikereallylongstrings?Ithoughtyoumightlikethemsoiaddedthistothetest")},
}

func TestAppendContainer(t *testing.T) {
	for _, c := range appendCases {
		buf := AppendContainer(nil, c)
		it := Iter(buf)
		assert.False(t, it.Done())
		{
			val, container, err := it.Next()
			assert.NoError(t, err)
			assert.True(t, container)
			containerIt := Iter(val)
			for _, expected := range c {
				assert.False(t, containerIt.Done())
				val, container, err := containerIt.Next()
				assert.NoError(t, err)
				assert.False(t, container)
				assert.Exactly(t, expected, val)
			}
			assert.True(t, containerIt.Done())
		}
		assert.True(t, it.Done())
	}
}

func TestAppendValue(t *testing.T) {
	for _, c := range appendCases {
		var buf []byte
		for _, val := range c {
			buf = AppendValue(buf, val)
		}
		it := Iter(buf)
		for _, expected := range c {
			assert.False(t, it.Done())
			val, container, err := it.Next()
			assert.NoError(t, err)
			assert.False(t, container)
			assert.Exactly(t, expected, val)
		}
		assert.True(t, it.Done())
	}
}

func TestUvarint(t *testing.T) {
	cases := []uint64{
		0,
		1,
		2,
		126,
		127,
		128,
		(127 << 7) + 126,
		(127 << 7) + 127,
		(127 << 7) + 128,
		math.MaxUint8 - 1,
		math.MaxUint8,
		math.MaxUint8 + 1,
		math.MaxUint16 - 1,
		math.MaxUint16,
		math.MaxUint16 + 1,
		math.MaxUint32 - 1,
		math.MaxUint32,
		math.MaxUint32 + 1,
		math.MaxUint64 - 2,
		math.MaxUint64 - 1,
		math.MaxUint64,
	}
	for _, c := range cases {
		buf := AppendUvarint(nil, c)
		u64, n := Uvarint(buf)
		require.Len(t, buf, n, "case: %d", c)
		require.Exactly(t, c, u64, "case: %d", c)

		buf = AppendUvarint(buf, c)
		u64, n = Uvarint(buf)
		require.Len(t, buf, n*2, "case: %d", c)
		require.Exactly(t, c, u64, "case: %d", c)
		u64, n = Uvarint(buf[n:])
		require.Len(t, buf, n*2, "case: %d", c)
		require.Exactly(t, c, u64, "case: %d", c)
	}
}
