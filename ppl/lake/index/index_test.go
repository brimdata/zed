package index

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
	"github.com/brimdata/zq/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteIndices(t *testing.T) {
	const data = `
{ts:1970-01-01T00:00:01Z,orig_h:127.0.0.1,proto:"conn"}
{ts:1970-01-01T00:00:02Z,orig_h:127.0.0.2,proto:"conn"}
`
	dir := iosrc.MustParseURI(t.TempDir())

	ctx := context.Background()
	r := zson.NewReader(strings.NewReader(data), zson.NewContext())

	ip := MustNewDefinition(NewTypeRule(zng.TypeIP))
	proto := MustNewDefinition(NewFieldRule("proto"))

	indices, err := WriteIndices(ctx, dir, r, ip, proto)
	require.NoError(t, err)
	assert.Len(t, indices, 2)

	tests := []struct {
		def     *Definition
		pattern string
		has     bool
	}{
		{
			def:     ip,
			pattern: "127.0.0.1",
			has:     true,
		},
		{
			def:     ip,
			pattern: "127.0.0.9",
			has:     false,
		},
		{
			def:     proto,
			pattern: "conn",
			has:     true,
		},
		{
			def:     proto,
			pattern: "http",
			has:     false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Find-%s-%t", test.def.Kind, test.has), func(t *testing.T) {
			reader, err := Find(ctx, resolver.NewContext(), dir, test.def.ID, test.pattern)
			require.NoError(t, err)
			recs, err := zbuf.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())

			if test.has {
				assert.Lenf(t, recs, 1, "expected query %s=%s to return a result", test.def, test.pattern)
			} else {
				assert.Lenf(t, recs, 0, "expected query %s=%s to return no result", test.def, test.pattern)
			}
		})
	}
}

func TestFindTypeRule(t *testing.T) {
	r := NewTypeRule(zng.TypeInt64)
	w := testWriter(t, r)
	err := zbuf.Copy(w, babbleReader(t))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	reader, err := FindFromPath(context.Background(), resolver.NewContext(), w.URI, "456")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Len(t, recs, 1)
	rec := recs[0]
	count, err := rec.AccessInt("count")
	require.NoError(t, err)
	key, err := rec.AccessInt("key")
	require.NoError(t, err)
	assert.EqualValues(t, 456, key)
	assert.EqualValues(t, 3, count)
}

func TestZQLRule(t *testing.T) {
	r, err := NewZqlRule("sum(v) by s | put key=s | sort key", "custom", nil)
	require.NoError(t, err)
	w := testWriter(t, r)
	err = zbuf.Copy(w, babbleReader(t))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	reader, err := FindFromPath(context.Background(), resolver.NewContext(), w.URI, "kartometer-trifocal")
	require.NoError(t, err)
	recs, err := zbuf.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Len(t, recs, 1)
	rec := recs[0]
	count, err := rec.AccessInt("sum")
	require.NoError(t, err)
	key, err := rec.AccessString("key")
	require.NoError(t, err)
	assert.EqualValues(t, "kartometer-trifocal", key)
	assert.EqualValues(t, 397, count)
}

func babbleReader(t *testing.T) zbuf.Reader {
	t.Helper()
	r, err := os.Open("../../../ztests/suite/data/babble-sorted.zson")
	require.NoError(t, err)
	return zson.NewReader(r, zson.NewContext())
}
