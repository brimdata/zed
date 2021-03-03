package index

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteIndices(t *testing.T) {
	const data = `
#0:record[ts:time,orig_h:ip,proto:string]
0:[1;127.0.0.1;conn;]
0:[2;127.0.0.2;conn;]
`
	dir := iosrc.MustParseURI(t.TempDir())

	ctx := context.Background()
	r := tzngio.NewReader(strings.NewReader(data), resolver.NewContext())

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
	r, err := os.Open("../../../ztests/suite/data/babble-sorted.tzng")
	require.NoError(t, err)
	return tzngio.NewReader(r, resolver.NewContext())
}
